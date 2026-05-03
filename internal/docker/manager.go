package docker

import (
	"fmt"
	"slices"

	"github.com/monlor/clash-gateway/internal/state"
)

type NetworkConnector interface {
	ConnectNetworkBackground(network, containerID string) error
}

type TrafficRedirector interface {
	Apply(pid int, gatewayIP string) error
	ApplyGateway() error
}

type Manager struct {
	GatewayName    string
	ManagedNetwork string
	Connector      NetworkConnector
	Redirector     TrafficRedirector
}

func DetectGatewayNameConflicts(containers []Container) error {
	owners := map[string]string{}
	for _, container := range containers {
		name := container.Labels[LabelManagedGatewayName]
		if name == "" {
			continue
		}
		if previous, ok := owners[name]; ok && previous != container.Name {
			return fmt.Errorf("gateway name conflict for %q between %q and %q", name, previous, container.Name)
		}
		owners[name] = container.Name
	}
	return nil
}

func (m Manager) SyncOnce(containers []Container) (state.Status, error) {
	if err := DetectGatewayNameConflicts(containers); err != nil {
		return state.Status{}, err
	}

	desired := BuildDesiredState(m.GatewayName, m.ManagedNetwork, containers)
	status := state.Status{
		GatewayName:    m.GatewayName,
		ManagedNetwork: m.ManagedNetwork,
	}
	gatewayIP := findGatewayIP(m.GatewayName, m.ManagedNetwork, containers)
	if m.Redirector != nil && gatewayIP != "" {
		if err := m.Redirector.ApplyGateway(); err != nil {
			return state.Status{}, err
		}
	}
	for _, container := range desired.Attach {
		if m.Connector != nil {
			if err := m.Connector.ConnectNetworkBackground(m.ManagedNetwork, container.ID); err != nil {
				return state.Status{}, err
			}
		}
		if m.Redirector != nil && container.PID > 0 && gatewayIP != "" {
			if err := m.Redirector.Apply(container.PID, gatewayIP); err != nil {
				return state.Status{}, err
			}
		}
		status.AttachedContainers = append(status.AttachedContainers, container.Name)
	}
	for _, container := range desired.Managed {
		if !hasNetwork(container.Networks, m.ManagedNetwork) {
			continue
		}
		if m.Redirector != nil && container.PID > 0 && gatewayIP != "" {
			if err := m.Redirector.Apply(container.PID, gatewayIP); err != nil {
				return state.Status{}, err
			}
		}
		status.AttachedContainers = append(status.AttachedContainers, container.Name)
	}
	for _, container := range desired.Pending {
		status.PendingContainers = append(status.PendingContainers, container.Name)
	}
	for _, rejected := range desired.Rejected {
		status.RejectedContainers = append(status.RejectedContainers, state.RejectedContainer{
			Name:   rejected.Container.Name,
			Reason: rejected.Reason,
		})
	}
	slices.Sort(status.AttachedContainers)
	slices.Sort(status.PendingContainers)
	slices.SortFunc(status.RejectedContainers, func(a, b state.RejectedContainer) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return status, nil
}

func findGatewayIP(gatewayName, managedNetwork string, containers []Container) string {
	for _, container := range containers {
		if container.Labels[LabelManagedGatewayName] != gatewayName {
			continue
		}
		if ip := container.NetworkIPs[managedNetwork]; ip != "" {
			return ip
		}
	}
	return ""
}
