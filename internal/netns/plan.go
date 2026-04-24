package netns

import (
	"fmt"
	"net"
)

const (
	ManagedMangleChain = "CLASH_GATEWAY_TPROXY"
	ManagedDivertChain = "CLASH_GATEWAY_DIVERT"
	DefaultRedirPort   = 7892
	DefaultTProxyPort  = 7893
	DefaultDNSPort     = 53
)

type Plan struct {
	Commands []string
}

func BuildGatewayPlan(tproxyPort int) Plan {
	if tproxyPort <= 0 || tproxyPort > 65535 {
		tproxyPort = DefaultTProxyPort
	}

	return Plan{
		Commands: []string{
			"sysctl -w net.ipv4.ip_forward=1",
			"ip rule show | grep -Eq \"fwmark 0x1 .* lookup 100|fwmark 0x1 lookup 100\" || ip rule add fwmark 1 table 100",
			"ip route replace local 0.0.0.0/0 dev lo table 100",
			fmt.Sprintf("iptables -t mangle -N %s 2>/dev/null || true", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -F %s", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -d 127.0.0.0/8 -j RETURN", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -d 10.0.0.0/8 -j RETURN", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -d 172.16.0.0/12 -j RETURN", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -d 192.168.0.0/16 -j RETURN", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -d 224.0.0.0/4 -j RETURN", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -j RETURN -m mark --mark 0xff", ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -A %s -p udp -j TPROXY --on-ip 127.0.0.1 --on-port %d --tproxy-mark 1", ManagedMangleChain, tproxyPort),
			fmt.Sprintf("iptables -t mangle -A %s -p tcp -j TPROXY --on-ip 127.0.0.1 --on-port %d --tproxy-mark 1", ManagedMangleChain, tproxyPort),
			fmt.Sprintf("iptables -t mangle -C PREROUTING -j %s 2>/dev/null || iptables -t mangle -A PREROUTING -j %s", ManagedMangleChain, ManagedMangleChain),
			fmt.Sprintf("iptables -t mangle -N %s 2>/dev/null || true", ManagedDivertChain),
			fmt.Sprintf("iptables -t mangle -F %s", ManagedDivertChain),
			fmt.Sprintf("iptables -t mangle -A %s -j MARK --set-mark 1", ManagedDivertChain),
			fmt.Sprintf("iptables -t mangle -A %s -j ACCEPT", ManagedDivertChain),
			fmt.Sprintf("iptables -t mangle -C PREROUTING -p tcp -m socket -j %s 2>/dev/null || iptables -t mangle -I PREROUTING -p tcp -m socket -j %s", ManagedDivertChain, ManagedDivertChain),
		},
	}
}

func BuildRoutePlan(gatewayIP string) (Plan, error) {
	if gatewayIP == "" {
		return Plan{}, fmt.Errorf("gateway IP is required")
	}
	if ip := net.ParseIP(gatewayIP); ip == nil {
		return Plan{}, fmt.Errorf("invalid gateway IP: %q", gatewayIP)
	}

	return Plan{
		Commands: []string{
			fmt.Sprintf("iface=$(ip route get %s | awk '{for(i=1;i<=NF;i++) if($i==\"dev\"){print $(i+1); exit}}'); [ -n \"$iface\" ] && ip route replace default via %s dev \"$iface\"", gatewayIP, gatewayIP),
			fmt.Sprintf("iptables -t nat -C OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:%d 2>/dev/null || iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:%d", gatewayIP, DefaultDNSPort, gatewayIP, DefaultDNSPort),
			fmt.Sprintf("iptables -t nat -C OUTPUT -p tcp --dport 53 -j DNAT --to-destination %s:%d 2>/dev/null || iptables -t nat -A OUTPUT -p tcp --dport 53 -j DNAT --to-destination %s:%d", gatewayIP, DefaultDNSPort, gatewayIP, DefaultDNSPort),
		},
	}, nil
}
