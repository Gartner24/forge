package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const DefaultAddr = "http://localhost:2019"

type Client struct {
	addr string
	http *http.Client
}

func New(addr string) *Client {
	if addr == "" {
		addr = DefaultAddr
	}
	return &Client{addr: addr, http: &http.Client{}}
}

type route struct {
	ID       string      `json:"@id"`
	Match    []hostMatch `json:"match"`
	Handle   []rpHandler `json:"handle"`
	Terminal bool        `json:"terminal"`
}

type hostMatch struct {
	Host []string `json:"host"`
}

type rpHandler struct {
	Handler   string     `json:"handler"`
	ID        string     `json:"@id,omitempty"`
	Upstreams []upstream `json:"upstreams"`
}

type upstream struct {
	Dial string `json:"dial"`
}

func routeID(projectID string) string  { return "smeltforge-" + projectID }
func rpID(projectID string) string     { return "smeltforge-" + projectID + "-rp" }

// AddRoute adds a reverse-proxy route for the project.
func (c *Client) AddRoute(projectID, domain, upstreamDial string) error {
	r := route{
		ID:    routeID(projectID),
		Match: []hostMatch{{Host: []string{domain}}},
		Handle: []rpHandler{{
			Handler:   "reverse_proxy",
			ID:        rpID(projectID),
			Upstreams: []upstream{{Dial: upstreamDial}},
		}},
		Terminal: true,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return c.do("POST", "/config/apps/http/servers/srv0/routes", data)
}

// UpdateUpstream atomically switches the upstream -- used for blue-green route swap.
func (c *Client) UpdateUpstream(projectID, upstreamDial string) error {
	h := rpHandler{
		Handler:   "reverse_proxy",
		ID:        rpID(projectID),
		Upstreams: []upstream{{Dial: upstreamDial}},
	}
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	return c.do("PATCH", "/id/"+rpID(projectID), data)
}

// RemoveRoute removes the route for the project.
func (c *Client) RemoveRoute(projectID string) error {
	return c.do("DELETE", "/id/"+routeID(projectID), nil)
}

// EnsureServer initialises Caddy's HTTP server config if it isn't present yet.
func (c *Client) EnsureServer() error {
	_, err := c.getRaw("/config/apps/http/servers/srv0")
	if err == nil {
		return nil
	}
	cfg := map[string]any{
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{
					"srv0": map[string]any{
						"listen": []string{":80", ":443"},
						"routes": []any{},
					},
				},
			},
			"tls": map[string]any{},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return c.do("POST", "/config/", data)
}

// Ping checks that the Caddy admin API is reachable.
func (c *Client) Ping() error {
	_, err := c.getRaw("/config/")
	return err
}

func (c *Client) getRaw(path string) ([]byte, error) {
	resp, err := c.http.Get(c.addr + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("caddy GET %s: %d %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) do(method, path string, body []byte) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.addr+path, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("caddy %s %s: %d %s", method, path, resp.StatusCode, string(respBody))
	}
	return nil
}
