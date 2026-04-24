package sandbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	ts := httptest.NewServer(handler)
	client := NewClientWithToken("test-token", ts.URL)
	return ts, client
}

func TestNewClientMissingToken(t *testing.T) {
	t.Setenv("DO_API_TOKEN", "")
	t.Setenv("DIGITALOCEAN_TOKEN", "")
	_, err := NewClient()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestNewClientWithDOAPIToken(t *testing.T) {
	t.Setenv("DO_API_TOKEN", "test-tok")
	t.Setenv("DIGITALOCEAN_TOKEN", "")
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "test-tok" {
		t.Errorf("expected test-tok, got %s", c.token)
	}
}

func TestNewClientFallsBackToDigitalOceanToken(t *testing.T) {
	t.Setenv("DO_API_TOKEN", "")
	t.Setenv("DIGITALOCEAN_TOKEN", "fallback-tok")
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "fallback-tok" {
		t.Errorf("expected fallback-tok, got %s", c.token)
	}
}

func TestCreateDroplet(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v2/droplets" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing auth header")
		}

		var body CreateDropletRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "test-sandbox" {
			t.Errorf("expected name test-sandbox, got %s", body.Name)
		}
		if body.Image != "ubuntu-24-04-x64" {
			t.Errorf("expected default image, got %s", body.Image)
		}

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"droplet": map[string]interface{}{
				"id":     12345,
				"name":   body.Name,
				"status": "new",
			},
		})
	})
	defer ts.Close()

	droplet, err := client.CreateDroplet(CreateDropletRequest{
		Name:   "test-sandbox",
		Region: "nyc1",
		Size:   "s-1vcpu-1gb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if droplet.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", droplet.ID)
	}
	if droplet.Name != "test-sandbox" {
		t.Errorf("expected name test-sandbox, got %s", droplet.Name)
	}
}

func TestGetDroplet(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/droplets/999" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"droplet": map[string]interface{}{
				"id":     999,
				"name":   "my-droplet",
				"status": "active",
				"networks": map[string]interface{}{
					"v4": []map[string]interface{}{
						{"ip_address": "10.0.0.1", "type": "private"},
						{"ip_address": "1.2.3.4", "type": "public"},
					},
				},
			},
		})
	})
	defer ts.Close()

	droplet, err := client.GetDroplet(999)
	if err != nil {
		t.Fatal(err)
	}
	if droplet.Status != "active" {
		t.Errorf("expected active, got %s", droplet.Status)
	}
	if droplet.PublicIP() != "1.2.3.4" {
		t.Errorf("expected public IP 1.2.3.4, got %s", droplet.PublicIP())
	}
}

func TestListDropletsWithTag(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tag_name") != "spekk-sandbox" {
			t.Errorf("expected tag_name query param, got %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"droplets": []map[string]interface{}{
				{"id": 1, "name": "sb-1", "status": "active"},
				{"id": 2, "name": "sb-2", "status": "new"},
			},
		})
	})
	defer ts.Close()

	droplets, err := client.ListDroplets("spekk-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if len(droplets) != 2 {
		t.Fatalf("expected 2 droplets, got %d", len(droplets))
	}
}

func TestDeleteDroplet(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v2/droplets/42" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	})
	defer ts.Close()

	if err := client.DeleteDroplet(42); err != nil {
		t.Fatal(err)
	}
}

func TestListSSHKeys(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/account/keys" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ssh_keys": []map[string]interface{}{
				{"id": 1, "name": "key1", "fingerprint": "aa:bb:cc"},
				{"id": 2, "name": "key2", "fingerprint": "dd:ee:ff"},
			},
		})
	})
	defer ts.Close()

	keys, err := client.ListSSHKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestCreateSSHKey(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "my-key" {
			t.Errorf("expected name my-key, got %s", body["name"])
		}

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ssh_key": map[string]interface{}{
				"id":          10,
				"name":        "my-key",
				"fingerprint": "xx:yy:zz",
			},
		})
	})
	defer ts.Close()

	key, err := client.CreateSSHKey("my-key", "ssh-rsa AAAA...")
	if err != nil {
		t.Fatal(err)
	}
	if key.Fingerprint != "xx:yy:zz" {
		t.Errorf("expected fingerprint xx:yy:zz, got %s", key.Fingerprint)
	}
}

func TestFindSSHKeyByFingerprint(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ssh_keys": []map[string]interface{}{
				{"id": 1, "name": "key1", "fingerprint": "aa:bb:cc"},
				{"id": 2, "name": "key2", "fingerprint": "dd:ee:ff"},
			},
		})
	})
	defer ts.Close()

	key, err := client.FindSSHKeyByFingerprint("dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	if key == nil {
		t.Fatal("expected to find key")
	}
	if key.ID != 2 {
		t.Errorf("expected key ID 2, got %d", key.ID)
	}

	// Not found
	notFound, err := client.FindSSHKeyByFingerprint("xx:xx:xx")
	if err != nil {
		t.Fatal(err)
	}
	if notFound != nil {
		t.Error("expected nil for not found key")
	}
}

func TestDeleteSSHKey(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v2/account/keys/42" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	err := client.DeleteSSHKey(42)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAssignToProject(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v2/projects/proj-123/resources" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		resources, _ := body["resources"].([]interface{})
		if len(resources) != 1 {
			t.Errorf("expected 1 resource, got %d", len(resources))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"resources": resources})
	})
	defer ts.Close()

	err := client.AssignToProject("proj-123", []string{"do:droplet:12345"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "unauthorized",
			"message": "Unable to authenticate you",
		})
	})
	defer ts.Close()

	_, err := client.GetDroplet(1)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Unable to authenticate you" {
		t.Errorf("unexpected message: %s", apiErr.Message)
	}
}

func TestAPIErrorNotFound(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "not_found",
			"message": "The resource you requested could not be found.",
		})
	})
	defer ts.Close()

	_, err := client.GetDroplet(99999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestDropletPublicIPEmpty(t *testing.T) {
	d := &Droplet{
		Networks: Networks{V4: []NetworkV4{
			{IPAddress: "10.0.0.1", Type: "private"},
		}},
	}
	if d.PublicIP() != "" {
		t.Error("expected empty public IP")
	}
}

func TestListProjects(t *testing.T) {
	ts, client := testServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"projects": []map[string]interface{}{
				{"id": "proj-1", "name": "My Project"},
			},
		})
	})
	defer ts.Close()

	projects, err := client.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Name != "My Project" {
		t.Errorf("expected 'My Project', got %s", projects[0].Name)
	}
}
