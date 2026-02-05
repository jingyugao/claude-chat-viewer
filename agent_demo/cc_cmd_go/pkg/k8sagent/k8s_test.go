package k8sagent

import "testing"

func TestSummarize(t *testing.T) {
	list := &ServiceList{Items: []Service{
		{
			Metadata: Metadata{Name: "svc-a", Namespace: "default"},
			Spec: ServiceSpec{
				Type:      "LoadBalancer",
				ClusterIP: "10.0.0.1",
			},
			Status: ServiceStatus{},
		},
		{
			Metadata: Metadata{Name: "svc-b", Namespace: "default"},
			Spec: ServiceSpec{
				Type:      "ClusterIP",
				ClusterIP: "None",
				Selector:  map[string]string{"app": "b"},
			},
		},
		{
			Metadata: Metadata{Name: "svc-c", Namespace: "ops"},
			Spec: ServiceSpec{
				Type:      "NodePort",
				ClusterIP: "10.0.0.3",
				Ports:     []ServicePort{{Port: 80, NodePort: 0}},
			},
		},
		{
			Metadata: Metadata{Name: "svc-d", Namespace: "ops"},
			Spec: ServiceSpec{
				Type:         "ExternalName",
				ExternalName: "example.com",
			},
		},
	}}

	s := Summarize(list)
	if s.Total != 4 {
		t.Fatalf("total = %d, want 4", s.Total)
	}
	if s.ByType["LoadBalancer"] != 1 || s.ByType["ClusterIP"] != 1 || s.ByType["NodePort"] != 1 || s.ByType["ExternalName"] != 1 {
		t.Fatalf("by type mismatch: %#v", s.ByType)
	}
	if len(s.Headless) != 1 {
		t.Fatalf("headless = %v", s.Headless)
	}
	if len(s.LoadBalancerPending) != 1 {
		t.Fatalf("lb pending = %v", s.LoadBalancerPending)
	}
	if len(s.NodePortMissing) != 1 {
		t.Fatalf("nodeport missing = %v", s.NodePortMissing)
	}
	if len(s.MissingSelector) != 2 {
		t.Fatalf("missing selector = %v", s.MissingSelector)
	}
}
