package openviking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"openIntern/internal/config"
)

// Manager OpenViking 进程管理器
type Manager struct {
	mu            sync.Mutex
	process       *exec.Cmd
	configPath    string
	config        *config.OpenVikingServiceConfig
	status        string
	lastError     string
	startTime     time.Time
	restartPolicy RestartPolicy
}

// RestartPolicy 重启策略
type RestartPolicy struct {
	Enabled         bool          // 是否自动重启
	MaxRetries      int           // 最大重试次数
	RetryDelay      time.Duration // 重试延迟
	HealthCheckURL  string        // 健康检查URL
	HealthCheckIntv time.Duration // 健康检查间隔
}

// StatusResponse 状态响应
type StatusResponse struct {
	Running   bool      `json:"running"`
	Status    string    `json:"status"`
	Pid       int       `json:"pid,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

// 全局管理器实例
var globalManager *Manager
var managerMu sync.Mutex

// InitManager 初始化 OpenViking 管理器
func InitManager(cfg *config.OpenVikingServiceConfig, configPath string) *Manager {
	managerMu.Lock()
	defer managerMu.Unlock()

	if globalManager == nil {
		globalManager = &Manager{
			configPath: configPath,
			config:     cfg,
			status:     "stopped",
			restartPolicy: RestartPolicy{
				Enabled:         true,
				MaxRetries:      3,
				RetryDelay:      5 * time.Second,
				HealthCheckIntv: 30 * time.Second,
			},
		}
	}
	return globalManager
}

// GetManager 获取全局管理器实例
func GetManager() *Manager {
	managerMu.Lock()
	defer managerMu.Unlock()
	return globalManager
}

// Start 启动 OpenViking 服务
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.process != nil && m.isProcessRunning() {
		return fmt.Errorf("openviking is already running")
	}

	// 查找 openviking-server 可执行文件
	execPath, err := findOpenVikingExecutable()
	if err != nil {
		m.status = "error"
		m.lastError = fmt.Sprintf("failed to find openviking-server: %v", err)
		return err
	}

	// 确保配置文件存在
	if err := m.ensureConfigFile(); err != nil {
		m.status = "error"
		m.lastError = fmt.Sprintf("failed to ensure config file: %v", err)
		return err
	}

	// 创建命令，使用 --config 参数指定配置文件
	ctx, cancel := context.WithCancel(context.Background())
	m.process = exec.CommandContext(ctx, execPath, "--config", m.configPath)

	// 设置进程组，便于终止
	m.process.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 重定向输出
	m.process.Stdout = os.Stdout
	m.process.Stderr = os.Stderr

	// 启动进程
	if err := m.process.Start(); err != nil {
		m.status = "error"
		m.lastError = fmt.Sprintf("failed to start openviking: %v", err)
		cancel()
		return err
	}

	m.status = "running"
	m.startTime = time.Now()
	m.lastError = ""

	log.Printf("openviking started with pid %d", m.process.Process.Pid)

	// 异步监控进程
	go m.monitorProcess(cancel)

	return nil
}

// Stop 停止 OpenViking 服务
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.process == nil || !m.isProcessRunning() {
		m.status = "stopped"
		return nil
	}

	// 发送 SIGTERM 信号
	if err := m.process.Process.Signal(syscall.SIGTERM); err != nil {
		// 如果 SIGTERM 失败，尝试 SIGKILL
		m.process.Process.Kill()
	}

	// 等待进程结束
	done := make(chan error, 1)
	go func() {
		done <- m.process.Wait()
	}()

	select {
	case <-done:
		log.Printf("openviking stopped")
	case <-time.After(10 * time.Second):
		// 超时强制终止
		m.process.Process.Kill()
		log.Printf("openviking force killed after timeout")
	}

	m.status = "stopped"
	m.process = nil
	return nil
}

// Restart 重启 OpenViking 服务
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}
	// 等待一秒确保完全停止
	time.Sleep(1 * time.Second)
	return m.Start()
}

// Status 获取 OpenViking 服务状态
func (m *Manager) Status() StatusResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	resp := StatusResponse{
		Status:    m.status,
		LastError: m.lastError,
	}

	if m.process != nil && m.isProcessRunning() {
		resp.Running = true
		resp.Pid = m.process.Process.Pid
		resp.StartTime = m.startTime
		resp.Uptime = time.Since(m.startTime).Round(time.Second).String()
	}

	return resp
}

// UpdateConfig 更新 OpenViking 配置
func (m *Manager) UpdateConfig(cfg *config.OpenVikingServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = cfg

	// 保存配置到文件
	if err := m.saveConfigFile(); err != nil {
		return err
	}

	return nil
}

// isProcessRunning 检查进程是否运行中
func (m *Manager) isProcessRunning() bool {
	if m.process == nil || m.process.Process == nil {
		return false
	}
	// 发送信号 0 检查进程是否存在
	err := m.process.Process.Signal(syscall.Signal(0))
	return err == nil
}

// monitorProcess 监控进程状态
func (m *Manager) monitorProcess(cancel context.CancelFunc) {
	err := m.process.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		m.lastError = err.Error()
		log.Printf("openviking process exited with error: %v", err)
	} else {
		log.Printf("openviking process exited normally")
	}

	m.status = "stopped"
	cancel()
}

// ensureConfigFile 确保配置文件存在
func (m *Manager) ensureConfigFile() error {
	// 检查文件是否存在
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return m.saveConfigFile()
	}
	return nil
}

// saveConfigFile 保存配置到文件
func (m *Manager) saveConfigFile() error {
	// 确保目录存在
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// findOpenVikingExecutable 查找 openviking-server 可执行文件
func findOpenVikingExecutable() (string, error) {
	// 尝试从 PATH 查找
	if path, err := exec.LookPath("openviking-server"); err == nil {
		return path, nil
	}

	// 尝试常见位置
	commonPaths := []string{
		"/usr/local/bin/openviking-server",
		"/usr/bin/openviking-server",
		filepath.Join(os.Getenv("HOME"), ".local/bin/openviking-server"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("openviking-server not found in PATH or common locations")
}

// maskAPIKey 脱敏 API Key（复用 config 包的函数）
func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:3] + "***" + key[len(key)-3:]
}