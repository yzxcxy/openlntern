package sandbox

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"openIntern/internal/config"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type DockerProvider struct {
	image             string
	host              string
	network           string
	healthcheckTimout time.Duration
}

type dockerInspectEntry struct {
	ID    string `json:"Id"`
	State struct {
		Running bool   `json:"Running"`
		Status  string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

// NewDockerProvider 创建本地 Docker CLI provider。
func NewDockerProvider(cfg config.SandboxConfig) (*DockerProvider, error) {
	image := strings.TrimSpace(cfg.Docker.Image)
	if image == "" {
		return nil, errors.New("tools.sandbox.docker.image is required")
	}
	host := strings.TrimSpace(cfg.Docker.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	return &DockerProvider{
		image:             image,
		host:              host,
		network:           strings.TrimSpace(cfg.Docker.Network),
		healthcheckTimout: time.Duration(cfg.HealthcheckTimeoutSeconds) * time.Second,
	}, nil
}

func (p *DockerProvider) Name() string {
	return ProviderDocker
}

func (p *DockerProvider) Create(ctx context.Context, req CreateRequest) (*CreateResult, error) {
	if existing, err := p.FindExisting(ctx, req.UserID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	if err := p.removeStaleContainer(ctx, req.UserID); err != nil {
		return nil, err
	}

	args := []string{
		"run",
		"-d",
		"--security-opt", "seccomp=unconfined",
		"--name", containerNameForUser(req.UserID),
		"--label", "openintern.managed=true",
		"--label", "openintern.user_id=" + strings.TrimSpace(req.UserID),
		"-p", "0:8080",
	}
	if p.network != "" {
		args = append(args, "--network", p.network)
	}
	args = append(args, p.image)

	output, err := dockerOutput(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "is already in use") {
			return p.FindExisting(ctx, req.UserID)
		}
		return nil, err
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return nil, errors.New("docker run returned empty container id")
	}

	return p.inspectResult(ctx, containerID)
}

func (p *DockerProvider) FindExisting(ctx context.Context, userID string) (*CreateResult, error) {
	containerID, err := p.findContainerID(ctx, userID)
	if err != nil || containerID == "" {
		return nil, err
	}
	return p.inspectResult(ctx, containerID)
}

func (p *DockerProvider) Destroy(ctx context.Context, instance Instance) error {
	target := strings.TrimSpace(instance.InstanceID)
	if target == "" {
		target = containerNameForUser(instance.UserID)
	}
	_, err := dockerOutput(ctx, "rm", "-f", target)
	return err
}

func (p *DockerProvider) HealthCheck(ctx context.Context, instance Instance) error {
	checkCtx, cancel := healthcheckContext(ctx, p.healthcheckTimout)
	defer cancel()

	mcpURL := strings.TrimRight(strings.TrimSpace(instance.Endpoint), "/") + "/mcp"
	var lastErr error
	for {
		if err := probeSandboxMCP(checkCtx, mcpURL); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if checkCtx.Err() != nil {
			break
		}
		timer := time.NewTimer(300 * time.Millisecond)
		select {
		case <-checkCtx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}

	if lastErr == nil {
		lastErr = checkCtx.Err()
	}
	if lastErr == nil {
		lastErr = errors.New("sandbox mcp health check timed out")
	}
	return fmt.Errorf("sandbox mcp health check failed: %w", lastErr)
}

func (p *DockerProvider) removeStaleContainer(ctx context.Context, userID string) error {
	containerID, err := p.findContainerID(ctx, userID)
	if err != nil || containerID == "" {
		return err
	}
	_, err = dockerOutput(ctx, "rm", "-f", containerID)
	return err
}

func (p *DockerProvider) findContainerID(ctx context.Context, userID string) (string, error) {
	output, err := dockerOutput(
		ctx,
		"ps",
		"-aq",
		"--filter", "label=openintern.managed=true",
		"--filter", "label=openintern.user_id="+strings.TrimSpace(userID),
	)
	if err != nil {
		return "", err
	}
	lines := strings.Fields(strings.TrimSpace(string(output)))
	if len(lines) == 0 {
		// label 不存在时再退回稳定容器名兜底。
		nameOutput, nameErr := dockerOutput(ctx, "ps", "-aq", "--filter", "name=^/"+containerNameForUser(userID)+"$")
		if nameErr != nil {
			return "", nameErr
		}
		lines = strings.Fields(strings.TrimSpace(string(nameOutput)))
	}
	if len(lines) == 0 {
		return "", nil
	}
	return lines[0], nil
}

func (p *DockerProvider) inspectResult(ctx context.Context, target string) (*CreateResult, error) {
	entry, err := p.inspectContainer(ctx, target)
	if err != nil {
		return nil, err
	}
	if !entry.State.Running {
		return nil, fmt.Errorf("sandbox container is not running: %s", entry.State.Status)
	}
	bindings := entry.NetworkSettings.Ports["8080/tcp"]
	if len(bindings) == 0 || strings.TrimSpace(bindings[0].HostPort) == "" {
		return nil, errors.New("sandbox container has no mapped host port")
	}
	return &CreateResult{
		InstanceID: entry.ID,
		Endpoint:   fmt.Sprintf("http://%s:%s", p.host, bindings[0].HostPort),
	}, nil
}

func (p *DockerProvider) inspectContainer(ctx context.Context, target string) (*dockerInspectEntry, error) {
	output, err := dockerOutput(ctx, "inspect", target)
	if err != nil {
		return nil, err
	}

	var entries []dockerInspectEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("docker inspect returned no entries")
	}
	return &entries[0], nil
}

func containerNameForUser(userID string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(userID)))
	return "openintern-sandbox-" + hex.EncodeToString(sum[:])[:12]
}

func dockerOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("docker %s failed: %s", strings.Join(args, " "), msg)
	}
	return output, nil
}

// probeSandboxMCP verifies that the sandbox MCP endpoint can finish one initialize handshake.
func probeSandboxMCP(ctx context.Context, mcpURL string) error {
	cli, err := client.NewStreamableHttpClient(mcpURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = cli.Close()
	}()

	if err := cli.Start(ctx); err != nil {
		return err
	}

	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{
		Name:    "openintern-sandbox-healthcheck",
		Version: "1.0.0",
	}
	_, err = cli.Initialize(ctx, request)
	return err
}
