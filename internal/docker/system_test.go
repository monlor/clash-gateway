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
