package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Ports struct {
	HTTPProxy          int
	SOCKSProxy         int
	ExternalController int
	UI                 int
}

type SubscriptionStatus struct {
	LastSuccessAt time.Time `json:"last_success_at,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
}

type Status struct {
	GatewayName        string             `json:"gateway_name"`
	ConfigMode         string             `json:"config_mode"`
	ManagedNetwork     string             `json:"managed_network"`
	Subscription       SubscriptionStatus `json:"subscription"`
	AttachedContainers []string           `json:"attached_containers"`
	PendingContainers  []string           `json:"pending_containers"`
	Ports              Ports              `json:"-"`
}

func (s Status) MarshalJSON() ([]byte, error) {
	type payload struct {
		GatewayName        string             `json:"gateway_name"`
		ConfigMode         string             `json:"config_mode"`
		ManagedNetwork     string             `json:"managed_network"`
		Subscription       SubscriptionStatus `json:"subscription"`
		AttachedContainers []string           `json:"attached_containers"`
		PendingContainers  []string           `json:"pending_containers"`
		HTTPProxyPort      int                `json:"http_proxy_port"`
		SOCKSProxyPort     int                `json:"socks_proxy_port"`
		ControllerPort     int                `json:"external_controller_port"`
		UIPort             int                `json:"ui_port"`
	}
	return json.Marshal(payload{
		GatewayName:        s.GatewayName,
		ConfigMode:         s.ConfigMode,
		ManagedNetwork:     s.ManagedNetwork,
		Subscription:       s.Subscription,
		AttachedContainers: s.AttachedContainers,
		PendingContainers:  s.PendingContainers,
		HTTPProxyPort:      s.Ports.HTTPProxy,
		SOCKSProxyPort:     s.Ports.SOCKSProxy,
		ControllerPort:     s.Ports.ExternalController,
		UIPort:             s.Ports.UI,
	})
}

func (s *Status) UnmarshalJSON(raw []byte) error {
	type payload struct {
		GatewayName        string             `json:"gateway_name"`
		ConfigMode         string             `json:"config_mode"`
		ManagedNetwork     string             `json:"managed_network"`
		Subscription       SubscriptionStatus `json:"subscription"`
		AttachedContainers []string           `json:"attached_containers"`
		PendingContainers  []string           `json:"pending_containers"`
		HTTPProxyPort      int                `json:"http_proxy_port"`
		SOCKSProxyPort     int                `json:"socks_proxy_port"`
		ControllerPort     int                `json:"external_controller_port"`
		UIPort             int                `json:"ui_port"`
	}

	var decoded payload
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	s.GatewayName = decoded.GatewayName
	s.ConfigMode = decoded.ConfigMode
	s.ManagedNetwork = decoded.ManagedNetwork
	s.Subscription = decoded.Subscription
	s.AttachedContainers = decoded.AttachedContainers
	s.PendingContainers = decoded.PendingContainers
	s.Ports = Ports{
		HTTPProxy:          decoded.HTTPProxyPort,
		SOCKSProxy:         decoded.SOCKSProxyPort,
		ExternalController: decoded.ControllerPort,
		UI:                 decoded.UIPort,
	}
	return nil
}

func Save(path string, status Status) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func Load(path string) (Status, error) {
	var status Status
	raw, err := os.ReadFile(path)
	if err != nil {
		return Status{}, err
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}
