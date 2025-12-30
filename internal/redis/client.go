package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ProxyState struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Port      int       `json:"port"`
	IP        string    `json:"ip"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
}

type Client struct {
	rdb *redis.Client
	ctx context.Context
}

func NewClient(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{rdb: rdb, ctx: ctx}, nil
}

func (c *Client) proxyKey(id int) string {
	return fmt.Sprintf("proxy:%d", id)
}

func (c *Client) SetProxyState(state *ProxyState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.rdb.Set(c.ctx, c.proxyKey(state.ID), data, 0).Err()
}

func (c *Client) GetProxyState(id int) (*ProxyState, error) {
	data, err := c.rdb.Get(c.ctx, c.proxyKey(id)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state ProxyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *Client) GetAllProxyStates(count int) ([]*ProxyState, error) {
	var states []*ProxyState
	for i := 1; i <= count; i++ {
		state, err := c.GetProxyState(i)
		if err != nil {
			continue
		}
		if state != nil {
			states = append(states, state)
		}
	}
	return states, nil
}

func (c *Client) GetHealthyProxies(count int) ([]*ProxyState, error) {
	all, _ := c.GetAllProxyStates(count)
	var healthy []*ProxyState
	for _, p := range all {
		if p.Healthy {
			healthy = append(healthy, p)
		}
	}
	return healthy, nil
}

func (c *Client) UpdateHealth(id int, healthy bool, ip string) error {
	state, _ := c.GetProxyState(id)
	if state == nil {
		state = &ProxyState{ID: id}
	}
	state.Healthy = healthy
	state.IP = ip
	state.LastCheck = time.Now()
	return c.SetProxyState(state)
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
