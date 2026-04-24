package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/docker"
	"github.com/monlor/clash-gateway/internal/gateway"
	"github.com/monlor/clash-gateway/internal/netns"
	"github.com/monlor/clash-gateway/internal/subscription"
)

func main() {
	cfg, err := config.ParseEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dockerCLI := docker.CLI{}
	service := gateway.Service{
		Config:     cfg,
		Fetch:      subscription.HTTPFetcher{}.Fetch,
		Connector:  dockerCLI,
		Redirector: netns.Redirector{},
		Containers: func() []gateway.ContainerSnapshot {
			containers, err := dockerCLI.Snapshot(context.Background())
			if err != nil {
				log.Printf("docker snapshot failed: %v", err)
				return nil
			}
			snapshots := make([]gateway.ContainerSnapshot, 0, len(containers))
			for _, container := range containers {
				snapshots = append(snapshots, gateway.ContainerSnapshot{
					ID:         container.ID,
					Name:       container.Name,
					PID:        container.PID,
					Labels:     container.Labels,
					Networks:   container.Networks,
					NetworkIPs: container.NetworkIPs,
				})
			}
			return snapshots
		},
	}

	result, err := service.Bootstrap()
	if err != nil {
		log.Fatal(err)
	}
	if raw, err := json.Marshal(result.Status); err == nil {
		log.Printf("startup status: %s", raw)
	}

	supervisor := newMihomoSupervisor(cfg.RuntimeConfigPath, cfg.DataDir)
	if err := supervisor.Start(); err != nil {
		log.Printf("mihomo start failed: %v", err)
	}

	var wg sync.WaitGroup
	startUIServer(ctx, &wg, cfg, controllerURL(cfg))

	refresh := func(reason string) {
		result, err := service.Refresh()
		if err != nil {
			log.Printf("%s refresh failed: %v", reason, err)
			return
		}
		if err := supervisor.Reload(); err != nil {
			log.Printf("mihomo reload failed after %s refresh: %v", reason, err)
		}
		if raw, err := json.Marshal(result.Status); err == nil {
			log.Printf("%s refresh status: %s", reason, raw)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh("periodic")
			}
		}
	}()

	if cfg.AutoUpdate && cfg.ConfigMode == config.ModeSubscription {
		startAutoRefresh(ctx, &wg, cfg, refresh)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dockerCLI.Watch(ctx, func(event docker.Event) {
			log.Printf("docker event %s/%s %s", event.Type, event.Action, event.Actor.ID)
			refresh("event")
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("docker watch failed: %v", err)
		}
	}()

	<-ctx.Done()
	if err := supervisor.Stop(); err != nil {
		log.Printf("mihomo stop failed: %v", err)
	}
	stop()
	wg.Wait()
}

func startAutoRefresh(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, refresh func(reason string)) {
	if cfg.UpdateCron != "" {
		scheduler := cron.New()
		_, err := scheduler.AddFunc(cfg.UpdateCron, func() {
			refresh("schedule")
		})
		if err != nil {
			log.Printf("invalid UPDATE_CRON: %v", err)
			return
		}
		scheduler.Start()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			stopCtx := scheduler.Stop()
			<-stopCtx.Done()
		}()
		return
	}

	interval, err := time.ParseDuration(cfg.UpdateInterval)
	if err != nil {
		log.Printf("invalid UPDATE_INTERVAL: %v", err)
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh("schedule")
			}
		}
	}()
}

func startUIServer(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, backendURL string) {
	if cfg.UIPort <= 0 {
		return
	}

	uiDir := filepath.Join(cfg.DataDir, "ui")
	if _, err := os.Stat(uiDir); err != nil {
		uiDir = "/opt/metacubexd"
	}
	_ = os.MkdirAll(uiDir, 0o755)
	configJS := "window.__METACUBEXD_CONFIG__ = { defaultBackendURL: window.location.origin + '/-/controller' }\n"
	_ = os.WriteFile(filepath.Join(uiDir, "config.js"), []byte(configJS), 0o644)

	target, err := url.Parse(backendURL)
	if err != nil {
		log.Printf("invalid controller url %q: %v", backendURL, err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/-/controller")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		if cfg.ControllerSecret != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.ControllerSecret)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/-/controller/", proxy)
	mux.Handle("/", http.FileServer(http.Dir(uiDir)))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.UIPort),
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("ui server failed: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}

func controllerURL(cfg config.Config) string {
	host := os.Getenv("EXTERNAL_CONTROLLER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, cfg.ExternalController)
}

type mihomoSupervisor struct {
	runtimeConfig string
	dataDir       string
	cmd           *exec.Cmd
}

func newMihomoSupervisor(runtimeConfig, dataDir string) *mihomoSupervisor {
	return &mihomoSupervisor{runtimeConfig: runtimeConfig, dataDir: dataDir}
}

func (s *mihomoSupervisor) Start() error {
	binary, err := resolveMihomoBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(binary, "-d", s.dataDir, "-f", s.runtimeConfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	s.cmd = cmd
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func (s *mihomoSupervisor) Reload() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Signal(syscall.SIGHUP)
}

func (s *mihomoSupervisor) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	return s.cmd.Process.Signal(syscall.SIGTERM)
}

func resolveMihomoBinary() (string, error) {
	candidates := []string{"/mihomo", "mihomo", "clash"}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "/") {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", errors.New("mihomo executable not found")
}
