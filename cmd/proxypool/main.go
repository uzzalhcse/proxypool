package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/uzzalhcse/proxypool/internal/api"
	"github.com/uzzalhcse/proxypool/internal/config"
	"github.com/uzzalhcse/proxypool/internal/docker"
	"github.com/uzzalhcse/proxypool/internal/proxy"
	"github.com/uzzalhcse/proxypool/internal/redis"
)

func main() {
	// Load .env
	godotenv.Load()
	godotenv.Load(".env")

	// Load config
	cfg := config.Load()
	log.Printf("[Main] Starting proxypool with %d WARP containers", cfg.WARPCount)

	// Connect to Redis
	redisClient, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("[Main] Redis connection failed: %v", err)
	}
	defer redisClient.Close()

	// Initialize Docker manager
	dockerMgr := docker.NewContainerManager(cfg)
	defer dockerMgr.StopAll()

	// Initialize proxy manager
	proxyMgr := proxy.NewManager(cfg, redisClient, dockerMgr)
	defer proxyMgr.Stop()

	// Wire up callback: register each proxy as it starts
	dockerMgr.OnProxyReady = func(id int, ip string) {
		proxyMgr.RegisterProxy(id, ip)
	}

	// Start load balancer IMMEDIATELY
	lb := proxy.NewLoadBalancer(cfg, proxyMgr, redisClient)
	go lb.Start()
	defer lb.Stop()

	// Start API server IMMEDIATELY
	apiHandler := api.NewHandler(cfg, proxyMgr, redisClient)
	go apiHandler.Start()

	// Wait for listeners to bind
	time.Sleep(100 * time.Millisecond)
	log.Printf("[Main] LB (:%d) and API (:%d) ready. Starting containers in background...", cfg.LBPort, cfg.APIPort)

	// Start health checker
	go proxyMgr.Start()

	// Start containers in BACKGROUND (each proxy registers via callback as it starts)
	go dockerMgr.StartAllWARPWithUniqueIPs()

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("\nShutting down...")
	log.Println("Goodbye!")
}
