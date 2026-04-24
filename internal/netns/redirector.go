package netns

import (
	"fmt"
	"net"
	"os/exec"
)

type Redirector struct {
	Exec func(name string, args ...string) error
}

func (r Redirector) Apply(pid int, gatewayIP string) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}
	if ip := net.ParseIP(gatewayIP); ip == nil {
		return fmt.Errorf("invalid gateway IP: %q", gatewayIP)
	}

	plan, err := BuildRoutePlan(gatewayIP)
	if err != nil {
		return err
	}
	for _, command := range plan.Commands {
		if err := r.exec("nsenter", "-t", fmt.Sprint(pid), "-n", "sh", "-c", command); err != nil {
			return err
		}
	}
	return nil
}

func (r Redirector) ApplyGateway() error {
	plan := BuildGatewayPlan(DefaultTProxyPort)
	for _, command := range plan.Commands {
		if err := r.exec("sh", "-c", command); err != nil {
			return err
		}
	}
	return nil
}

func (r Redirector) exec(name string, args ...string) error {
	if r.Exec != nil {
		return r.Exec(name, args...)
	}
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, string(output))
	}
	return nil
}
