package collector

import (
	"testing"

	"github.com/getbundled/getbundled-agent/internal/contracts"
)

func TestCollectOpenPortsShape(t *testing.T) {
	got := collectOpenPorts(5)
	if got == nil {
		t.Skip("net connections unavailable in this environment")
	}
	listening, ok := got["listening"].([]int)
	if !ok {
		t.Fatalf("listening ports type = %T, want []int", got["listening"])
	}
	for i := 1; i < len(listening); i++ {
		if listening[i] < listening[i-1] {
			t.Fatalf("ports not sorted: %v", listening)
		}
	}
}

func TestFailedServiceNames(t *testing.T) {
	services := []contracts.ServiceStatusSample{
		{Name: "docker", Active: true},
		{Name: "nginx", Active: false},
	}
	got := failedServiceNames(services)
	if len(got) != 1 || got[0] != "nginx" {
		t.Fatalf("failedServiceNames() = %#v, want [nginx]", got)
	}
}
