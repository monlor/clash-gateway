package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

type Event struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID string `json:"ID"`
	} `json:"Actor"`
}

type CLI struct {
	Command string
}

func (c CLI) Snapshot(ctx context.Context) ([]Container, error) {
	command := c.commandName()

	psCmd := exec.CommandContext(ctx, command, "ps", "-aq")
	ids, err := psCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}

	fields := strings.Fields(string(ids))
	if len(fields) == 0 {
		return nil, nil
	}

	args := append([]string{"inspect"}, fields...)
	inspectCmd := exec.CommandContext(ctx, command, args...)
	raw, err := inspectCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect: %w", err)
	}
	return parseInspectOutput(raw)
}

func (c CLI) ConnectNetwork(ctx context.Context, network, containerID string) error {
	command := c.commandName()
	cmd := exec.CommandContext(ctx, command, "network", "connect", network, containerID)
	output, err := cmd.CombinedOutput()
	if err != nil && !bytes.Contains(output, []byte("already exists")) {
		return fmt.Errorf("docker network connect: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (c CLI) ConnectNetworkBackground(network, containerID string) error {
	return c.ConnectNetwork(context.Background(), network, containerID)
}

func (c CLI) Watch(ctx context.Context, onEvent func(Event)) error {
	command := c.commandName()
	cmd := exec.CommandContext(ctx, command, "events", "--filter", "type=container", "--filter", "type=network", "--format", "{{json .}}")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		event, ok := parseEventLine(scanner.Bytes())
		if ok {
			onEvent(event)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func parseInspectOutput(raw []byte) ([]Container, error) {
	var payload []struct {
		ID    string `json:"Id"`
		Name  string `json:"Name"`
		State struct {
			Pid int `json:"Pid"`
		} `json:"State"`
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	containers := make([]Container, 0, len(payload))
	for _, item := range payload {
		networks := make([]string, 0, len(item.NetworkSettings.Networks))
		networkIPs := make(map[string]string, len(item.NetworkSettings.Networks))
		for name := range item.NetworkSettings.Networks {
			networks = append(networks, name)
			networkIPs[name] = item.NetworkSettings.Networks[name].IPAddress
		}
		slices.Sort(networks)
		containers = append(containers, Container{
			ID:         item.ID,
			Name:       strings.TrimPrefix(item.Name, "/"),
			PID:        item.State.Pid,
			Labels:     item.Config.Labels,
			Networks:   networks,
			NetworkIPs: networkIPs,
		})
	}
	return containers, nil
}

func parseEventLine(raw []byte) (Event, bool) {
	var event Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return Event{}, false
	}
	if event.Type == "" || event.Action == "" {
		return Event{}, false
	}
	return event, true
}

func (c CLI) commandName() string {
	if c.Command != "" {
		return c.Command
	}
	return "docker"
}
