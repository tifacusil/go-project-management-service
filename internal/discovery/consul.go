// Package discovery handles Consul service registration (CLAUDE.md §4, §6.5)
// via the agent HTTP API directly — no SDK, zero extra deps.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Registration holds enough state to deregister later.
type Registration struct {
	httpc      *http.Client
	baseURL    string // e.g. http://localhost:8500
	token      string
	instanceID string
}

// Options configures a single service registration.
type Options struct {
	ConsulAddr  string // host:port of the Consul agent
	Token       string // ACL token; empty if ACLs disabled
	ServiceName string // e.g. project-management-service
	InstanceID  string // e.g. project-management-service-<uuid>
	Address     string // advertised host/ip (prefer-ip-address)
	Port        int
	HealthURL   string   // full HTTP url Consul will poll, e.g. http://host:8084/health
	Interval    string   // health check interval, e.g. "10s"
	Tags        []string // convention: version-<v>, lang-go, env-<env>
}

// agentCheck mirrors the Consul agent check JSON.
type agentCheck struct {
	HTTP                           string `json:"HTTP"`
	Interval                       string `json:"Interval"`
	Timeout                        string `json:"Timeout"`
	DeregisterCriticalServiceAfter string `json:"DeregisterCriticalServiceAfter"`
}

// agentService mirrors the body of PUT /v1/agent/service/register.
type agentService struct {
	ID      string     `json:"ID"`
	Name    string     `json:"Name"`
	Address string     `json:"Address"`
	Port    int        `json:"Port"`
	Tags    []string   `json:"Tags"`
	Check   agentCheck `json:"Check"`
}

// Register issues PUT /v1/agent/service/register against the local Consul agent.
// Caller decides whether failure is fatal — baseline uses fail-fast=false.
func Register(opts Options) (*Registration, error) {
	body := agentService{
		ID:      opts.InstanceID,
		Name:    opts.ServiceName,
		Address: opts.Address,
		Port:    opts.Port,
		Tags:    opts.Tags,
		Check: agentCheck{
			HTTP:                           opts.HealthURL,
			Interval:                       opts.Interval,
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}

	reg := &Registration{
		httpc:      &http.Client{Timeout: 5 * time.Second},
		baseURL:    "http://" + opts.ConsulAddr,
		token:      opts.Token,
		instanceID: opts.InstanceID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := reg.put(ctx, "/v1/agent/service/register", body); err != nil {
		return nil, fmt.Errorf("consul register %s: %w", opts.InstanceID, err)
	}
	return reg, nil
}

// Deregister issues PUT /v1/agent/service/deregister/:id. Safe on nil receiver.
func (r *Registration) Deregister() error {
	if r == nil || r.httpc == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.put(ctx, "/v1/agent/service/deregister/"+r.instanceID, nil); err != nil {
		return fmt.Errorf("consul deregister %s: %w", r.instanceID, err)
	}
	return nil
}

// put sends a JSON PUT to the agent, attaching the ACL token if present.
func (r *Registration) put(ctx context.Context, path string, payload any) error {
	var rdr io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		rdr = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.token != "" {
		req.Header.Set("X-Consul-Token", r.token)
	}

	resp, err := r.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("consul agent %s: status %d: %s", path, resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}
