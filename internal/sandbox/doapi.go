// Package sandbox provides the DigitalOcean API client and sandbox management.
package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const baseURL = "https://api.digitalocean.com"

// Client is a DigitalOcean API client.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string // overridable for testing
}

// NewClient creates a new DigitalOcean API client.
// Uses DO_API_TOKEN or DIGITALOCEAN_TOKEN environment variable.
func NewClient() (*Client, error) {
	token := os.Getenv("DO_API_TOKEN")
	if token == "" {
		token = os.Getenv("DIGITALOCEAN_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf(
			"DO_API_TOKEN environment variable is not set. " +
				"Get a token from https://cloud.digitalocean.com/account/api/tokens",
		)
	}
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}, nil
}

// NewClientWithToken creates a client with an explicit token (for testing).
func NewClientWithToken(token, base string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    base,
	}
}

// --- Droplet types ---

// Droplet represents a DigitalOcean droplet.
type Droplet struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Region    Region    `json:"region"`
	Size      Size      `json:"size"`
	Image     Image     `json:"image"`
	Networks  Networks  `json:"networks"`
	Tags      []string  `json:"tags"`
	VpcUUID   string    `json:"vpc_uuid"`
	CreatedAt string    `json:"created_at"`
}

// Region represents a DO region.
type Region struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Size represents a DO droplet size.
type Size struct {
	Slug string `json:"slug"`
}

// Image represents a DO image.
type Image struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Networks contains droplet network info.
type Networks struct {
	V4 []NetworkV4 `json:"v4"`
}

// NetworkV4 represents an IPv4 network interface.
type NetworkV4 struct {
	IPAddress string `json:"ip_address"`
	Type      string `json:"type"` // "public" or "private"
}

// PublicIP returns the first public IPv4 address, or empty string.
func (d *Droplet) PublicIP() string {
	for _, n := range d.Networks.V4 {
		if n.Type == "public" {
			return n.IPAddress
		}
	}
	return ""
}

// CreateDropletRequest contains options for creating a droplet.
type CreateDropletRequest struct {
	Name      string   `json:"name"`
	Region    string   `json:"region"`
	Size      string   `json:"size"`
	Image     string   `json:"image"`
	SSHKeys   []int    `json:"ssh_keys,omitempty"`
	UserData  string   `json:"user_data,omitempty"`
	Tags      []string `json:"tags"`
	VpcUUID   string   `json:"vpc_uuid,omitempty"`
}

// SSHKey represents a DigitalOcean SSH key.
type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

// Project represents a DigitalOcean project.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- API error ---

// APIError represents an error from the DigitalOcean API.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("DigitalOcean API error: %s (%d %s)", e.Message, e.StatusCode, e.Status)
}

// --- Droplet operations ---

// CreateDroplet creates a new droplet.
func (c *Client) CreateDroplet(req CreateDropletRequest) (*Droplet, error) {
	if req.Image == "" {
		req.Image = "ubuntu-24-04-x64"
	}
	if req.Tags == nil {
		req.Tags = []string{"spekk-sandbox"}
	}

	var resp struct {
		Droplet *Droplet `json:"droplet"`
	}
	if err := c.doRequest("POST", "/v2/droplets", req, &resp); err != nil {
		return nil, err
	}
	return resp.Droplet, nil
}

// GetDroplet retrieves a droplet by ID.
func (c *Client) GetDroplet(id int) (*Droplet, error) {
	var resp struct {
		Droplet *Droplet `json:"droplet"`
	}
	if err := c.doRequest("GET", fmt.Sprintf("/v2/droplets/%d", id), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Droplet, nil
}

// ListDroplets lists droplets, optionally filtered by tag.
func (c *Client) ListDroplets(tag string) ([]Droplet, error) {
	path := "/v2/droplets"
	if tag != "" {
		path += "?tag_name=" + url.QueryEscape(tag)
	}
	var resp struct {
		Droplets []Droplet `json:"droplets"`
	}
	if err := c.doRequest("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Droplets, nil
}

// DeleteDroplet destroys a droplet by ID.
func (c *Client) DeleteDroplet(id int) error {
	return c.doRequest("DELETE", fmt.Sprintf("/v2/droplets/%d", id), nil, nil)
}

// --- SSH Key operations ---

// ListSSHKeys returns all SSH keys on the account.
func (c *Client) ListSSHKeys() ([]SSHKey, error) {
	var resp struct {
		SSHKeys []SSHKey `json:"ssh_keys"`
	}
	if err := c.doRequest("GET", "/v2/account/keys", nil, &resp); err != nil {
		return nil, err
	}
	return resp.SSHKeys, nil
}

// CreateSSHKey adds a new SSH key to the account.
func (c *Client) CreateSSHKey(name, publicKey string) (*SSHKey, error) {
	payload := map[string]string{
		"name":       name,
		"public_key": publicKey,
	}
	var resp struct {
		SSHKey *SSHKey `json:"ssh_key"`
	}
	if err := c.doRequest("POST", "/v2/account/keys", payload, &resp); err != nil {
		return nil, err
	}
	return resp.SSHKey, nil
}

// DeleteSSHKey removes an SSH key from the account by ID.
func (c *Client) DeleteSSHKey(id int) error {
	return c.doRequest("DELETE", fmt.Sprintf("/v2/account/keys/%d", id), nil, nil)
}

// FindSSHKeyByFingerprint searches for an SSH key by fingerprint.
func (c *Client) FindSSHKeyByFingerprint(fingerprint string) (*SSHKey, error) {
	keys, err := c.ListSSHKeys()
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if k.Fingerprint == fingerprint {
			return &k, nil
		}
	}
	return nil, nil
}

// --- Project operations ---

// ListProjects returns all projects on the account.
func (c *Client) ListProjects() ([]Project, error) {
	var resp struct {
		Projects []Project `json:"projects"`
	}
	if err := c.doRequest("GET", "/v2/projects", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Projects, nil
}

// AssignToProject assigns resources (by URN) to a project.
func (c *Client) AssignToProject(projectID string, resourceURNs []string) error {
	payload := map[string]interface{}{
		"resources": resourceURNs,
	}
	return c.doRequest("POST", fmt.Sprintf("/v2/projects/%s/resources", projectID), payload, nil)
}

// --- HTTP helper ---

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content (e.g., DELETE)
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract error message from response
		var apiResp struct {
			Message string `json:"message"`
			ID      string `json:"id"`
		}
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if json.Unmarshal(respBody, &apiResp) == nil {
			if apiResp.Message != "" {
				msg = apiResp.Message
			} else if apiResp.ID != "" {
				msg = apiResp.ID
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Message:    msg,
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
