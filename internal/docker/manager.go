package docker

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/uzzalhcse/proxypool/internal/config"
)

const (
	maxRetries    = 20
	retryDelay    = 5 * time.Second
	warpInitDelay = 10 * time.Second
)

type ContainerManager struct {
	cfg          *config.Config
	usedIPs      map[string]bool
	OnProxyReady func(id int, ip string) // Callback when a proxy is ready
}

func NewContainerManager(cfg *config.Config) *ContainerManager {
	return &ContainerManager{
		cfg:     cfg,
		usedIPs: make(map[string]bool),
	}
}

// StartAllWARPWithUniqueIPs starts all WARP containers serially, ensuring unique IPs
func (m *ContainerManager) StartAllWARPWithUniqueIPs() error {
	log.Printf("[Docker] Starting %d WARP containers with unique IPs...", m.cfg.WARPCount)

	for i := 1; i <= m.cfg.WARPCount; i++ {
		ip, err := m.startWARPWithUniqueIP(i)
		if err != nil {
			log.Printf("[Docker] Warning: Container warp-%d failed: %v", i, err)
			continue
		}
		m.usedIPs[ip] = true
		log.Printf("[Docker] ✓ warp-%d ready with IP %s", i, ip)

		// Notify immediately so LB can use this proxy
		if m.OnProxyReady != nil {
			m.OnProxyReady(i, ip)
		}
	}

	log.Printf("[Docker] All containers started. Unique IPs: %v", m.getUsedIPsList())
	return nil
}

// startWARPWithUniqueIP starts a single WARP container, retrying until unique IP
func (m *ContainerManager) startWARPWithUniqueIP(id int) (string, error) {
	name := fmt.Sprintf("warp-%d", id)
	port := m.cfg.WARPBasePort + id - 1

	for retry := 1; retry <= maxRetries; retry++ {
		log.Printf("[Docker] Starting %s (attempt %d/%d)...", name, retry, maxRetries)

		// Remove any existing container + volumes (forces new registration)
		exec.Command("docker", "rm", "-f", "-v", name).Run()
		time.Sleep(1 * time.Second)

		// Start fresh container
		if err := m.runWARPContainer(name, port); err != nil {
			log.Printf("[Docker] Failed to start %s: %v", name, err)
			continue
		}

		// Wait for WARP to initialize
		time.Sleep(warpInitDelay)

		// Get IP
		ip, err := m.GetContainerIP(port)
		if err != nil {
			log.Printf("[Docker] Failed to get IP for %s: %v", name, err)
			continue
		}

		// Check if unique
		if !m.usedIPs[ip] {
			return ip, nil
		}

		log.Printf("[Docker] Duplicate IP %s for %s, retrying...", ip, name)
		time.Sleep(retryDelay)
	}

	return "", fmt.Errorf("failed to get unique IP after %d retries", maxRetries)
}

// runWARPContainer runs a WARP container
func (m *ContainerManager) runWARPContainer(name string, port int) error {
	args := []string{
		"run", "-d",
		"--name", name,
		"--cap-add", "NET_ADMIN",
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"-p", fmt.Sprintf("%d:9091", port),
		"monius/docker-warp-socks:latest",
	}

	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// GetContainerIP gets the external IP through the proxy
func (m *ContainerManager) GetContainerIP(port int) (string, error) {
	proxy := fmt.Sprintf("socks5://127.0.0.1:%d", port)
	cmd := exec.Command("curl", "-x", proxy, "-s", "--max-time", "15", "https://api.ipify.org")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("empty IP response")
	}
	return ip, nil
}

// GetUsedIPs returns the current used IPs map
func (m *ContainerManager) GetUsedIPs() map[string]bool {
	return m.usedIPs
}

func (m *ContainerManager) getUsedIPsList() []string {
	ips := []string{}
	for ip := range m.usedIPs {
		ips = append(ips, ip)
	}
	return ips
}

// StopAll stops all WARP containers
func (m *ContainerManager) StopAll() {
	for i := 1; i <= m.cfg.WARPCount; i++ {
		exec.Command("docker", "rm", "-f", fmt.Sprintf("warp-%d", i)).Run()
	}
}

// RestartWithNewIP restarts a container to get a new unique IP
func (m *ContainerManager) RestartWithNewIP(id int, currentIP string) (string, error) {
	// Remove current IP from used list
	delete(m.usedIPs, currentIP)

	// Get new unique IP
	ip, err := m.startWARPWithUniqueIP(id)
	if err != nil {
		// Restore old IP on failure
		m.usedIPs[currentIP] = true
		return "", err
	}

	m.usedIPs[ip] = true
	return ip, nil
}
