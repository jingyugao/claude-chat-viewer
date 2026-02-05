package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type serviceList struct {
	Items []service `json:"items"`
}

type service struct {
	Metadata metadata     `json:"metadata"`
	Spec     serviceSpec  `json:"spec"`
	Status   serviceStatus `json:"status"`
}

type metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type serviceSpec struct {
	Type         string            `json:"type"`
	ClusterIP    string            `json:"clusterIP"`
	ExternalName string            `json:"externalName"`
	Selector     map[string]string `json:"selector"`
	Ports        []servicePort     `json:"ports"`
	ExternalIPs  []string          `json:"externalIPs"`
}

type servicePort struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	NodePort int32  `json:"nodePort"`
	Protocol string `json:"protocol"`
}

type serviceStatus struct {
	LoadBalancer loadBalancerStatus `json:"loadBalancer"`
}

type loadBalancerStatus struct {
	Ingress []lbIngress `json:"ingress"`
}

type lbIngress struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

func main() {
	var kubeconfig string
	var kubeContext string
	var namespace string
	var timeout time.Duration
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	flag.StringVar(&kubeContext, "context", "", "Kube context")
	flag.StringVar(&namespace, "namespace", "", "Namespace filter (default all)")
	flag.DurationVar(&timeout, "timeout", 20*time.Second, "kubectl timeout")
	flag.Parse()

	list, err := fetchServices(kubeconfig, kubeContext, namespace, timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	report(list, namespace)
}

func fetchServices(kubeconfig, kubeContext, namespace string, timeout time.Duration) (*serviceList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	args := []string{"get", "svc", "-o", "json"}
	if namespace == "" {
		args = append(args, "-A")
	} else {
		args = append(args, "-n", namespace)
	}
	if kubeconfig != "" {
		args = append([]string{"--kubeconfig", kubeconfig}, args...)
	}
	if kubeContext != "" {
		args = append([]string{"--context", kubeContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("kubectl timed out after %s", timeout)
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("kubectl failed: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}

	var list serviceList
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("parse kubectl json: %w", err)
	}
	return &list, nil
}

func report(list *serviceList, namespace string) {
	total := len(list.Items)
	byType := make(map[string]int)
	byNamespace := make(map[string]int)

	var headless []service
	var missingSelector []service
	var lbPending []service
	var nodePortMissing []service

	for _, svc := range list.Items {
		stype := svc.Spec.Type
		if stype == "" {
			stype = "ClusterIP"
		}
		byType[stype]++
		byNamespace[svc.Metadata.Namespace]++

		if strings.EqualFold(svc.Spec.ClusterIP, "None") {
			headless = append(headless, svc)
		}
		if len(svc.Spec.Selector) == 0 && stype != "ExternalName" {
			missingSelector = append(missingSelector, svc)
		}
		if stype == "LoadBalancer" {
			external := len(svc.Spec.ExternalIPs) > 0 || len(svc.Status.LoadBalancer.Ingress) > 0
			if !external {
				lbPending = append(lbPending, svc)
			}
		}
		if stype == "NodePort" {
			missing := false
			for _, p := range svc.Spec.Ports {
				if p.NodePort == 0 {
					missing = true
					break
				}
			}
			if missing {
				nodePortMissing = append(nodePortMissing, svc)
			}
		}
	}

	fmt.Printf("Total services: %d\n", total)
	fmt.Println("By type:")
	printMap(byType)

	if namespace == "" {
		fmt.Println("Top namespaces:")
		printTopN(byNamespace, 10)
	}

	if len(headless) > 0 {
		fmt.Printf("Headless services: %d (clusterIP=None)\n", len(headless))
	}
	if len(missingSelector) > 0 {
		fmt.Printf("Services without selector (non-ExternalName): %d\n", len(missingSelector))
		printSample(missingSelector, 10)
	}
	if len(lbPending) > 0 {
		fmt.Printf("LoadBalancer pending external IP: %d\n", len(lbPending))
		printSample(lbPending, 10)
	}
	if len(nodePortMissing) > 0 {
		fmt.Printf("NodePort services missing nodePort: %d\n", len(nodePortMissing))
		printSample(nodePortMissing, 10)
	}
}

func printMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("- %s: %d\n", k, m[k])
	}
}

func printTopN(m map[string]int, n int) {
	type kv struct {
		Key string
		Val int
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{Key: k, Val: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Val == items[j].Val {
			return items[i].Key < items[j].Key
		}
		return items[i].Val > items[j].Val
	})
	if len(items) < n {
		n = len(items)
	}
	for i := 0; i < n; i++ {
		fmt.Printf("- %s: %d\n", items[i].Key, items[i].Val)
	}
}

func printSample(services []service, n int) {
	if len(services) < n {
		n = len(services)
	}
	for i := 0; i < n; i++ {
		s := services[i]
		fmt.Printf("- %s/%s\n", s.Metadata.Namespace, s.Metadata.Name)
	}
}
