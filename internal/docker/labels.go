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
	ID          string
	Name        string
	PID         int
	Labels      map[string]string
	Networks    []string
	NetworkIPs  map[string]string
	NetworkMode string
}

type RejectedContainer struct {
	Container Container
	Reason    string
}

type DesiredState struct {
	Managed  []Container
	Attach   []Container
	Pending  []Container
	Rejected []RejectedContainer
}

const HostNetworkRejectReason = "host network containers cannot be routed through clash-gateway"

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
		if isHostNetworkMode(container.NetworkMode) {
			desired.Rejected = append(desired.Rejected, RejectedContainer{
				Container: container,
				Reason:    HostNetworkRejectReason,
			})
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

func isHostNetworkMode(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "host")
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
