package docker

import "strings"

const (
	LabelGateway            = "clash-gateway.gateway"
	LabelAllowAttach        = "clash-gateway.allow-attach"
	LabelDisable            = "clash-gateway.disable"
	LabelManagedGatewayName = "clash-gateway.name"
	LabelAttachNetworkName  = "clash-gateway.attach-network"
)

type Target struct {
	GatewayName string
	AllowAttach bool
	Disabled    bool
}

type Container struct {
	ID         string
	Name       string
	PID        int
	Labels     map[string]string
	Networks   []string
	NetworkIPs map[string]string
}

type DesiredState struct {
	Managed []Container
	Attach  []Container
	Pending []Container
}

func ParseTarget(labels map[string]string) (Target, bool) {
	gatewayName := strings.TrimSpace(labels[LabelGateway])
	if gatewayName == "" {
		return Target{}, false
	}
	return Target{
		GatewayName: gatewayName,
		AllowAttach: parseBool(labels[LabelAllowAttach]),
		Disabled:    parseBool(labels[LabelDisable]),
	}, true
}

func BuildDesiredState(gatewayName, managedNetwork string, containers []Container) DesiredState {
	var desired DesiredState
	for _, container := range containers {
		target, ok := ParseTarget(container.Labels)
		if !ok || target.Disabled || target.GatewayName != gatewayName {
			continue
		}

		desired.Managed = append(desired.Managed, container)
		if hasNetwork(container.Networks, managedNetwork) {
			continue
		}
		if target.AllowAttach {
			desired.Attach = append(desired.Attach, container)
			continue
		}
		desired.Pending = append(desired.Pending, container)
	}
	return desired
}

func hasNetwork(networks []string, target string) bool {
	for _, network := range networks {
		if network == target {
			return true
		}
	}
	return false
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
