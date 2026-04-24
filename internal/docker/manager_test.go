package docker_test

import (
	"testing"

	"github.com/monlor/clash-gateway/internal/docker"
)

type fakeConnector struct {
	calls []struct {
		network string
		id      string
	}
}

func (f *fakeConnector) ConnectNetworkBackground(network, containerID string) error {
	f.calls = append(f.calls, struct {
		network string
		id      string
	}{network: network, id: containerID})
	return nil
}

type fakeRedirector struct {
	gatewayCalls int
	calls        []struct {
		pid       int
		gatewayIP string
	}
}

func (f *fakeRedirector) ApplyGateway() error {
	f.gatewayCalls++
	return nil
}

func (f *fakeRedirector) Apply(pid int, gatewayIP string) error {
	f.calls = append(f.calls, struct {
		pid       int
		gatewayIP string
	}{pid: pid, gatewayIP: gatewayIP})
	return nil
}

func TestDetectGatewayNameConflicts(t *testing.T) {
	t.Parallel()

	err := docker.DetectGatewayNameConflicts([]docker.Container{
		{
			Name: "gateway-a",
			Labels: map[string]string{
				docker.LabelManagedGatewayName: "hk",
			},
		},
		{
			Name: "gateway-b",
			Labels: map[string]string{
				docker.LabelManagedGatewayName: "hk",
			},
		},
	})
	if err == nil {
		t.Fatal("DetectGatewayNameConflicts returned nil error, want conflict error")
	}
}

func TestSyncOnceProducesAttachedAndPendingNames(t *testing.T) {
	t.Parallel()

	connector := &fakeConnector{}
	redirector := &fakeRedirector{}
	manager := docker.Manager{
		GatewayName:    "hk",
		ManagedNetwork: "clash-gateway-hk",
		Connector:      connector,
		Redirector:     redirector,
	}
	status, err := manager.SyncOnce([]docker.Container{
		{
			Name: "gateway-hk",
			Labels: map[string]string{
				docker.LabelManagedGatewayName: "hk",
			},
			Networks:   []string{"clash-gateway-hk"},
			NetworkIPs: map[string]string{"clash-gateway-hk": "172.20.0.2"},
		},
		{
			ID:   "a1",
			Name: "app-a",
			PID:  1001,
			Labels: map[string]string{
				docker.LabelGateway:     "hk",
				docker.LabelAllowAttach: "true",
			},
			Networks: []string{"bridge", "clash-gateway-hk"},
		},
		{
			ID:   "b2",
			Name: "app-b",
			PID:  1002,
			Labels: map[string]string{
				docker.LabelGateway: "hk",
			},
			Networks: []string{"bridge"},
		},
	})
	if err != nil {
		t.Fatalf("SyncOnce returned error: %v", err)
	}
	if len(status.AttachedContainers) != 1 || status.AttachedContainers[0] != "app-a" {
		t.Fatalf("AttachedContainers = %#v, want [app-a]", status.AttachedContainers)
	}
	if redirector.gatewayCalls != 1 {
		t.Fatalf("redirector.gatewayCalls = %d, want 1", redirector.gatewayCalls)
	}
	if len(status.PendingContainers) != 1 || status.PendingContainers[0] != "app-b" {
		t.Fatalf("PendingContainers = %#v, want [app-b]", status.PendingContainers)
	}
	if len(connector.calls) != 0 {
		t.Fatalf("connector.calls = %#v, want no attach because app-a is already on network", connector.calls)
	}
	if len(redirector.calls) != 1 || redirector.calls[0].pid != 1001 || redirector.calls[0].gatewayIP != "172.20.0.2" {
		t.Fatalf("redirector.calls = %#v, want one redirect for app-a via 172.20.0.2", redirector.calls)
	}
}

func TestSyncOnceAttachesThenRedirectsContainer(t *testing.T) {
	t.Parallel()

	connector := &fakeConnector{}
	redirector := &fakeRedirector{}
	manager := docker.Manager{
		GatewayName:    "hk",
		ManagedNetwork: "clash-gateway-hk",
		Connector:      connector,
		Redirector:     redirector,
	}

	_, err := manager.SyncOnce([]docker.Container{
		{
			Name: "gateway-hk",
			Labels: map[string]string{
				docker.LabelManagedGatewayName: "hk",
			},
			Networks:   []string{"clash-gateway-hk"},
			NetworkIPs: map[string]string{"clash-gateway-hk": "172.20.0.2"},
		},
		{
			ID:   "a1",
			Name: "app-a",
			PID:  1001,
			Labels: map[string]string{
				docker.LabelGateway:     "hk",
				docker.LabelAllowAttach: "true",
			},
			Networks: []string{"bridge"},
		},
	})
	if err != nil {
		t.Fatalf("SyncOnce returned error: %v", err)
	}
	if len(connector.calls) != 1 || connector.calls[0].id != "a1" {
		t.Fatalf("connector.calls = %#v, want one attach for app-a", connector.calls)
	}
	if redirector.gatewayCalls != 1 {
		t.Fatalf("redirector.gatewayCalls = %d, want 1", redirector.gatewayCalls)
	}
	if len(redirector.calls) != 1 || redirector.calls[0].pid != 1001 {
		t.Fatalf("redirector.calls = %#v, want one redirect for app-a", redirector.calls)
	}
}
