package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uzzalhcse/proxypool/internal/config"
	"github.com/uzzalhcse/proxypool/internal/redis"
)

type LoadBalancer struct {
	cfg     *config.Config
	redis   *redis.Client
	manager *Manager
	stopCh  chan struct{}

	proxies []*redis.ProxyState
	cacheMu sync.RWMutex
	counter uint64
}

func NewLoadBalancer(cfg *config.Config, manager *Manager, redis *redis.Client) *LoadBalancer {
	return &LoadBalancer{
		cfg:     cfg,
		redis:   redis,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
}

func (lb *LoadBalancer) Start() error {
	lb.refreshCache()
	go lb.cacheRefresher()
	go lb.runListener()

	log.Printf("[LB] Started on :%d (cache refresh: 5s)", lb.cfg.LBPort)
	return nil
}

func (lb *LoadBalancer) Stop() {
	log.Printf("[LB] Stopping...")
	close(lb.stopCh)
}

func (lb *LoadBalancer) cacheRefresher() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lb.refreshCache()
		case <-lb.stopCh:
			return
		}
	}
}

func (lb *LoadBalancer) refreshCache() {
	proxies, _ := lb.manager.GetHealthyProxies()
	lb.cacheMu.Lock()
	oldCount := len(lb.proxies)
	lb.proxies = proxies
	lb.cacheMu.Unlock()

	if len(proxies) != oldCount {
		log.Printf("[LB] Cache refreshed: %d healthy proxies", len(proxies))
	}
}

func (lb *LoadBalancer) runListener() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", lb.cfg.LBPort))
	if err != nil {
		log.Printf("[LB] ✗ Failed to listen: %v", err)
		return
	}
	defer listener.Close()

	for {
		select {
		case <-lb.stopCh:
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go lb.handleConnection(conn)
	}
}

func (lb *LoadBalancer) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Get next proxy (round-robin)
	proxy := lb.getNextProxy()
	if proxy == nil {
		log.Printf("[LB] ✗ No healthy proxies available")
		return
	}

	// Connect to upstream WARP proxy
	upstreamAddr := fmt.Sprintf("127.0.0.1:%d", proxy.Port)
	upstreamConn, err := net.DialTimeout("tcp", upstreamAddr, 10*time.Second)
	if err != nil {
		log.Printf("[LB] ✗ Upstream connection failed: %v", err)
		return
	}
	defer upstreamConn.Close()

	// Pipe data both directions
	errCh := make(chan error, 2)
	go func() { _, err := io.Copy(upstreamConn, clientConn); errCh <- err }()
	go func() { _, err := io.Copy(clientConn, upstreamConn); errCh <- err }()

	<-errCh
}

func (lb *LoadBalancer) getNextProxy() *redis.ProxyState {
	lb.cacheMu.RLock()
	defer lb.cacheMu.RUnlock()

	if len(lb.proxies) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&lb.counter, 1) % uint64(len(lb.proxies))
	return lb.proxies[idx]
}

func (lb *LoadBalancer) Stats() map[string]interface{} {
	lb.cacheMu.RLock()
	defer lb.cacheMu.RUnlock()

	return map[string]interface{}{
		"healthy_count": len(lb.proxies),
		"port":          lb.cfg.LBPort,
	}
}
