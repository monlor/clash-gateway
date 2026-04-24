package docker

import (
	"testing"
)

func TestParseInspectOutput(t *testing.T) {
	t.Parallel()

	raw := []byte(`[
	  {
	    "Id": "abc123",
	    "Name": "/app-a",
	    "State": {
	      "Pid": 4321
	    },
	    "Config": {
	      "Labels": {
	        "clash-gateway.gateway": "hk",
	        "clash-gateway.allow-attach": "true"
	      }
	    },
	    "NetworkSettings": {
	      "Networks": {
	        "bridge": {
	          "IPAddress": "172.17.0.3"
	        },
	        "clash-gateway-hk": {
	          "IPAddress": "172.20.0.10"
	        }
	      }
	    }
	  }
	]`)

	containers, err := parseInspectOutput(raw)
	if err != nil {
		t.Fatalf("parseInspectOutput returned error: %v", err)
	}
	if len(containers) != 1 {
		t.Fatalf("len(containers) = %d, want 1", len(containers))
	}
	if containers[0].Name != "app-a" {
		t.Fatalf("Name = %q, want app-a", containers[0].Name)
	}
	if containers[0].PID != 4321 {
		t.Fatalf("PID = %d, want 4321", containers[0].PID)
	}
	if len(containers[0].Networks) != 2 {
		t.Fatalf("Networks = %#v, want two networks", containers[0].Networks)
	}
	if containers[0].NetworkIPs["clash-gateway-hk"] != "172.20.0.10" {
		t.Fatalf("NetworkIPs = %#v, want managed IP", containers[0].NetworkIPs)
	}
}

func TestParseEventLine(t *testing.T) {
	t.Parallel()

	event, ok := parseEventLine([]byte(`{"Type":"container","Action":"start","Actor":{"ID":"abc123"}}`))
	if !ok {
		t.Fatal("parseEventLine returned ok=false, want true")
	}
	if event.Type != "container" || event.Action != "start" || event.Actor.ID != "abc123" {
		t.Fatalf("event = %#v, want parsed event", event)
	}
}

func TestFindSelfGatewayNameByContainerIDPrefix(t *testing.T) {
	t.Parallel()

	containers := []Container{
		{
			ID: "abc123456789",
			Labels: map[string]string{
				LabelManagedGatewayName: "main",
			},
		},
	}

	name, err := FindSelfGatewayName(containers, "abc123456789")
	if err != nil {
		t.Fatalf("FindSelfGatewayName returned error: %v", err)
	}
	if name != "main" {
		t.Fatalf("gateway name = %q, want main", name)
	}
}

func TestFindSelfGatewayNameRejectsMissingManagedLabel(t *testing.T) {
	t.Parallel()

	containers := []Container{
		{ID: "abc123456789", Labels: map[string]string{}},
	}

	if _, err := FindSelfGatewayName(containers, "abc123456789"); err == nil {
		t.Fatal("FindSelfGatewayName returned nil error, want label error")
	}
}

func TestFindSelfManagedNetworkUsesAttachNetworkLabel(t *testing.T) {
	t.Parallel()

	containers := []Container{
		{
			ID: "abc123456789",
			Labels: map[string]string{
				LabelManagedGatewayName: "main",
				LabelAttachNetworkName:  "custom-net",
			},
		},
	}

	network, err := FindSelfManagedNetwork(containers, "abc123456789", "main")
	if err != nil {
		t.Fatalf("FindSelfManagedNetwork returned error: %v", err)
	}
	if network != "custom-net" {
		t.Fatalf("managed network = %q, want custom-net", network)
	}
}

func TestFindSelfManagedNetworkFallsBackToDefault(t *testing.T) {
	t.Parallel()

	containers := []Container{
		{
			ID: "abc123456789",
			Labels: map[string]string{
				LabelManagedGatewayName: "main",
			},
		},
	}

	network, err := FindSelfManagedNetwork(containers, "abc123456789", "main")
	if err != nil {
		t.Fatalf("FindSelfManagedNetwork returned error: %v", err)
	}
	if network != "clash-gateway-main" {
		t.Fatalf("managed network = %q, want clash-gateway-main", network)
	}
}
