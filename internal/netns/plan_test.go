package netns_test

import (
	"strings"
	"testing"

	"github.com/monlor/clash-gateway/internal/netns"
)

func TestBuildGatewayPlanIncludesTProxyRules(t *testing.T) {
	t.Parallel()

	plan := netns.BuildGatewayPlan(7893)
	if len(plan.Commands) == 0 {
		t.Fatal("BuildGatewayPlan returned no commands")
	}

	wantFragments := []string{
		"sysctl -w net.ipv4.ip_forward=1",
		"ip rule add fwmark 1 table 100",
		"ip route replace local 0.0.0.0/0 dev lo table 100",
		"iptables -t mangle -N CLASH_GATEWAY_TPROXY",
		"TPROXY --on-ip 127.0.0.1 --on-port 7893 --tproxy-mark 1",
		"iptables -t mangle -A PREROUTING -j CLASH_GATEWAY_TPROXY",
		"iptables -t mangle -I PREROUTING -p tcp -m socket -j CLASH_GATEWAY_DIVERT",
	}
	for _, fragment := range wantFragments {
		found := false
		for _, command := range plan.Commands {
			if strings.Contains(command, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("plan.Commands = %#v, want fragment %q", plan.Commands, fragment)
		}
	}
}

func TestBuildGatewayPlanFallsBackForInvalidPort(t *testing.T) {
	t.Parallel()

	plan := netns.BuildGatewayPlan(65536)

	command := strings.Join(plan.Commands, "\n")
	if !strings.Contains(command, "--on-port 7893 --tproxy-mark 1") {
		t.Fatalf("command = %q, want default tproxy port", command)
	}
}

func TestBuildRoutePlanReplacesDefaultGateway(t *testing.T) {
	t.Parallel()

	plan, err := netns.BuildRoutePlan("192.168.97.2")
	if err != nil {
		t.Fatalf("BuildRoutePlan returned error: %v", err)
	}
	if len(plan.Commands) != 3 {
		t.Fatalf("len(plan.Commands) = %d, want 3", len(plan.Commands))
	}

	// BuildRoutePlan intentionally returns three shell commands that contain
	// four relevant fragments when joined together for assertion readability.
	command := strings.Join(plan.Commands, "\n")
	for _, fragment := range []string{
		"ip route get 192.168.97.2",
		"ip route replace default via 192.168.97.2",
		"iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination 192.168.97.2:53",
		"iptables -t nat -A OUTPUT -p tcp --dport 53 -j DNAT --to-destination 192.168.97.2:53",
	} {
		if !strings.Contains(command, fragment) {
			t.Fatalf("command = %q, want fragment %q", command, fragment)
		}
	}
}

func TestBuildRoutePlanRejectsInvalidGatewayIP(t *testing.T) {
	t.Parallel()

	if _, err := netns.BuildRoutePlan("not-an-ip"); err == nil {
		t.Fatal("BuildRoutePlan accepted invalid gateway IP")
	}
}
