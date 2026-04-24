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
	    "Config": {
	      "Labels": {
	        "clash-gateway.gateway": "hk",
	        "clash-gateway.allow-attach": "true"
	      }
	    },
	    "NetworkSettings": {
	      "Networks": {
	        "bridge": {},
	        "clash-gateway-hk": {}
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
	if len(containers[0].Networks) != 2 {
		t.Fatalf("Networks = %#v, want two networks", containers[0].Networks)
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
