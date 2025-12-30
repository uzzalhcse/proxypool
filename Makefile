.PHONY: build run start stop status logs clean dev

# Build the proxypool binary
build:
	go build -o bin/proxypool ./cmd/proxypool

# Run proxypool (requires Redis to be running)
run: build
	./bin/proxypool

# Start Redis
start:
	docker compose -f docker/docker-compose.yml up -d

# Stop everything: proxypool process, WARP containers, and Redis
stop:
	@pkill proxypool 2>/dev/null || true
	@docker rm -f $$(docker ps -aq --filter "name=warp") 2>/dev/null || true
	docker compose -f docker/docker-compose.yml down

# Show container status
status:
	docker ps --filter "name=proxypool\|warp" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# View logs
logs:
	docker compose -f docker/docker-compose.yml logs -f

# Clean build artifacts and stop all WARP containers
clean:
	rm -rf bin/
	@docker rm -f $$(docker ps -aq --filter "name=warp") 2>/dev/null || true

# Full development: start Redis + run proxypool
dev: start
	sleep 3
	$(MAKE) run
