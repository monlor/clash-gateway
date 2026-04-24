package docker_test

import (
	"testing"

	"github.com/monlor/clash-gateway/internal/docker"
)

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

	manager := docker.Manager{
		GatewayName:    "hk",
		ManagedNetwork: "clash-gateway-hk",
	}
	status, err := manager.SyncOnce([]docker.Container{
		{
			Name: "gateway-hk",
			Labels: map[string]string{
				docker.LabelManagedGatewayName: "hk",
			},
		},
		{
			Name: "app-a",
			Labels: map[string]string{
				docker.LabelGateway:     "hk",
				docker.LabelAllowAttach: "true",
			},
			Networks: []string{"bridge"},
		},
		{
			Name: "app-b",
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
	if len(status.PendingContainers) != 1 || status.PendingContainers[0] != "app-b" {
		t.Fatalf("PendingContainers = %#v, want [app-b]", status.PendingContainers)
	}
}
