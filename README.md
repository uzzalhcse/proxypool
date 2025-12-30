# ProxyPool

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A high-performance WARP proxy pool with **guaranteed unique IPs** and round-robin load balancing.

## Features

- ✅ **Unique IPs** - Each container gets a different IP (retries until unique)
- ✅ **Immediate Availability** - LB and API start instantly, proxies register as they come up
- ✅ **Load Balancing** - Round-robin across all healthy proxies
- ✅ **Auto-Health Checks** - Monitors proxy health every 30s
- ✅ **Simple Config** - Just set `WARP_COUNT` and go

## Quick Start

```bash
git clone https://github.com/uzzalhcse/proxypool.git
cd proxypool

# Configure
cp .env.example .env
vim .env  # Set WARP_COUNT, credentials

# Start
make dev
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6380` | Redis connection |
| `WARP_COUNT` | `3` | Number of WARP proxies |
| `PROXY_AUTH_USER` | `testuser` | SOCKS5 username |
| `PROXY_AUTH_PASS` | `testpass123` | SOCKS5 password |
| `API_AUTH_TOKEN` | `secret123` | API Bearer token |
| `LB_PORT` | `40000` | Load balancer port |
| `API_PORT` | `8080` | API server port |
| `HEALTH_INTERVAL` | `30s` | Health check interval |

## Usage

### Load Balancer (Recommended)

```bash
# Round-robin across all proxies
curl -x socks5://user:pass@localhost:40000 https://api.ipify.org
```

### Direct Proxy Access

```bash
curl -x socks5://localhost:40001 https://api.ipify.org  # warp-1
curl -x socks5://localhost:40002 https://api.ipify.org  # warp-2
curl -x socks5://localhost:40003 https://api.ipify.org  # warp-3
```

## API

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /api/health` | No | Health check |
| `GET /api/proxies` | Yes | List all proxies |

```bash
curl http://localhost:8080/api/health
curl -H "Authorization: Bearer secret123" http://localhost:8080/api/proxies
```

## Port Reference

| Port | Service |
|------|---------|
| 40000 | Load balancer |
| 40001+ | WARP proxies |
| 8080 | REST API |
| 6380 | Redis |

## How Unique IPs Work

1. Containers start **serially** (one after another)
2. Each container retries up to 20 times with 5s delay
3. Only accepts IP not already used by another container
4. Each proxy is available in LB **immediately** after starting

## Architecture

```
                    ┌──────────────┐
                    │   Client     │
                    └──────┬───────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
        ┌──────────┐             ┌──────────┐
        │ LB:40000 │             │ API:8080 │
        └────┬─────┘             └──────────┘
             │
    ┌────────┼────────┐
    ▼        ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐
│warp-1 │ │warp-2 │ │warp-3 │
│:40001 │ │:40002 │ │:40003 │
└───────┘ └───────┘ └───────┘
```

## Development

```bash
make build    # Build binary
make run      # Run service
make dev      # Start Redis + run proxypool
make stop     # Stop everything
make clean    # Clean build + containers
make status   # Show container status
```

## Project Structure

```
proxypool/
├── cmd/proxypool/main.go     # Entry point
├── internal/
│   ├── api/handler.go        # REST API
│   ├── config/config.go      # Configuration
│   ├── docker/manager.go     # Container management
│   ├── proxy/
│   │   ├── loadbalancer.go   # SOCKS5 load balancer
│   │   └── manager.go        # Proxy state management
│   └── redis/client.go       # Redis persistence
├── docker/
│   └── docker-compose.yml    # Redis container
├── Makefile
└── README.md
```

## License

MIT
