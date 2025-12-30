package proxy

import (
	"log"
	"sync"
	"time"

	"github.com/uzzalhcse/proxypool/internal/config"
	"github.com/uzzalhcse/proxypool/internal/docker"
	"github.com/uzzalhcse/proxypool/internal/redis"
)

type Manager struct {
	cfg    *config.Config
	redis  *redis.Client
	docker *docker.ContainerManager
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewManager(cfg *config.Config, redisClient *redis.Client, dockerMgr *docker.ContainerManager) *Manager {
	return &Manager{
		cfg:    cfg,
		redis:  redisClient,
		docker: dockerMgr,
		stopCh: make(chan struct{}),
	}
}

// RegisterProxy registers a single proxy immediately (called as each container starts)
func (m *Manager) RegisterProxy(id int, ip string) {
	port := m.cfg.WARPBasePort + id - 1
	m.redis.SetProxyState(&redis.ProxyState{
		ID:        id,
		Type:      "warp",
		Port:      port,
		IP:        ip,
		Healthy:   true,
		LastCheck: time.Now(),
	})
	log.Printf("[Manager] ✓ Registered proxy %d (port %d) with IP %s", id, port, ip)
}

// Start starts the health checker
func (m *Manager) Start() error {
	log.Printf("[Manager] Starting health checker (interval: %s)", m.cfg.HealthInterval)
	m.wg.Add(1)
	go m.healthChecker()
	return nil
}

// Stop stops the manager
func (m *Manager) Stop() {
	log.Printf("[Manager] Stopping...")
	close(m.stopCh)
	m.wg.Wait()
	log.Printf("[Manager] Stopped")
}

// healthChecker periodically checks proxy health
func (m *Manager) healthChecker() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAllProxies()
		case <-m.stopCh:
			return
		}
	}
}

// checkAllProxies checks health of all proxies
func (m *Manager) checkAllProxies() {
	healthy, total := 0, 0
	for i := 1; i <= m.cfg.WARPCount; i++ {
		state, _ := m.redis.GetProxyState(i)
		if state == nil {
			continue
		}
		total++
		if m.checkProxy(state) {
			healthy++
		}
	}
	log.Printf("[Health] Check complete: %d/%d healthy", healthy, total)
}

// checkProxy checks a single proxy
func (m *Manager) checkProxy(state *redis.ProxyState) bool {
	ip, err := m.docker.GetContainerIP(state.Port)
	healthy := err == nil && ip != ""

	if healthy {
		m.redis.UpdateHealth(state.ID, true, ip)
		return true
	}
	log.Printf("[Health] ✗ Proxy %d DOWN (port %d)", state.ID, state.Port)
	m.redis.UpdateHealth(state.ID, false, "")
	return false
}

// GetHealthyProxies returns all healthy proxies
func (m *Manager) GetHealthyProxies() ([]*redis.ProxyState, error) {
	return m.redis.GetHealthyProxies(m.cfg.WARPCount)
}

// GetAllProxies returns all proxy states
func (m *Manager) GetAllProxies() ([]*redis.ProxyState, error) {
	return m.redis.GetAllProxyStates(m.cfg.WARPCount)
}
