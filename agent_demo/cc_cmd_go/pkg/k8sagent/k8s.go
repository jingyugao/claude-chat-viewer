package k8sagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type ServiceList struct {
	Items []Service `json:"items"`
}

type Service struct {
	Metadata Metadata      `json:"metadata"`
	Spec     ServiceSpec   `json:"spec"`
	Status   ServiceStatus `json:"status"`
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ServiceSpec struct {
	Type         string            `json:"type"`
	ClusterIP    string            `json:"clusterIP"`
	ExternalName string            `json:"externalName"`
	Selector     map[string]string `json:"selector"`
	Ports        []ServicePort     `json:"ports"`
	ExternalIPs  []string          `json:"externalIPs"`
}

type ServicePort struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	NodePort int32  `json:"nodePort"`
	Protocol string `json:"protocol"`
}

type ServiceStatus struct {
	LoadBalancer LoadBalancerStatus `json:"loadBalancer"`
}

type LoadBalancerStatus struct {
	Ingress []LBIngress `json:"ingress"`
}

type LBIngress struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

type Summary struct {
	Total               int
	ByType              map[string]int
	ByNamespace         map[string]int
	Headless            []string
	MissingSelector     []string
	LoadBalancerPending []string
	NodePortMissing     []string
}

func FetchServices(ctx context.Context, kubectlPath, kubeconfig, kubeContext, namespace string, timeout time.Duration) (*ServiceList, error) {
	if kubectlPath == "" {
		kubectlPath = "kubectl"
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
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

	cmd := exec.CommandContext(ctx, kubectlPath, args...)
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

	var list ServiceList
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("parse kubectl json: %w", err)
	}
	return &list, nil
}

func Summarize(list *ServiceList) Summary {
	summary := Summary{
		Total:       0,
		ByType:      map[string]int{},
		ByNamespace: map[string]int{},
	}
	if list == nil {
		return summary
	}
	for _, svc := range list.Items {
		summary.Total++
		stype := svc.Spec.Type
		if stype == "" {
			stype = "ClusterIP"
		}
		summary.ByType[stype]++
		summary.ByNamespace[svc.Metadata.Namespace]++

		name := svc.Metadata.Namespace + "/" + svc.Metadata.Name
		if strings.EqualFold(svc.Spec.ClusterIP, "None") {
			summary.Headless = append(summary.Headless, name)
		}
		if len(svc.Spec.Selector) == 0 && stype != "ExternalName" {
			summary.MissingSelector = append(summary.MissingSelector, name)
		}
		if stype == "LoadBalancer" {
			external := len(svc.Spec.ExternalIPs) > 0 || len(svc.Status.LoadBalancer.Ingress) > 0
			if !external {
				summary.LoadBalancerPending = append(summary.LoadBalancerPending, name)
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
				summary.NodePortMissing = append(summary.NodePortMissing, name)
			}
		}
	}
	return summary
}

func RenderSummary(summary Summary, namespace string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total services: %d\n", summary.Total))
	b.WriteString("By type:\n")
	b.WriteString(renderMap(summary.ByType))
	if namespace == "" {
		b.WriteString("Top namespaces:\n")
		b.WriteString(renderTopN(summary.ByNamespace, 10))
	}
	if len(summary.Headless) > 0 {
		b.WriteString(fmt.Sprintf("Headless services: %d (clusterIP=None)\n", len(summary.Headless)))
	}
	if len(summary.MissingSelector) > 0 {
		b.WriteString(fmt.Sprintf("Services without selector (non-ExternalName): %d\n", len(summary.MissingSelector)))
		b.WriteString(renderSample(summary.MissingSelector, 10))
	}
	if len(summary.LoadBalancerPending) > 0 {
		b.WriteString(fmt.Sprintf("LoadBalancer pending external IP: %d\n", len(summary.LoadBalancerPending)))
		b.WriteString(renderSample(summary.LoadBalancerPending, 10))
	}
	if len(summary.NodePortMissing) > 0 {
		b.WriteString(fmt.Sprintf("NodePort services missing nodePort: %d\n", len(summary.NodePortMissing)))
		b.WriteString(renderSample(summary.NodePortMissing, 10))
	}
	return strings.TrimSpace(b.String())
}

func renderMap(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- %s: %d\n", k, m[k]))
	}
	return b.String()
}

func renderTopN(m map[string]int, n int) string {
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
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(fmt.Sprintf("- %s: %d\n", items[i].Key, items[i].Val))
	}
	return b.String()
}

func renderSample(items []string, n int) string {
	if len(items) < n {
		n = len(items)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(fmt.Sprintf("- %s\n", items[i]))
	}
	return b.String()
}
