package sandbox

import (
	"errors"
	"strings"
	"testing"
)

// Provision finishes a record that Create left at "provisioning". It must
// refuse a record in any other state unless the operator forces it, and it
// must fail before touching the machine when a credential is missing.
func TestProvision(t *testing.T) {
	// save writes a record for a droplet spekk created.
	save := func(t *testing.T, status, auth string) {
		t.Helper()
		meta := &SandboxMeta{Provider: "digitalocean", DropletID: 42, IP: "1.2.3.4", Status: status, Auth: auth}
		if err := SaveSandbox("box", meta); err != nil {
			t.Fatal(err)
		}
	}
	// stubCheck replaces the marker check and reports whether it ran.
	stubCheck := func(t *testing.T, err error) *bool {
		t.Helper()
		checked := false
		orig := checkReady
		checkReady = func(meta *SandboxMeta, name string) error {
			checked = true
			return err
		}
		t.Cleanup(func() { checkReady = orig })
		return &checked
	}

	t.Run("refuses an active record without --force", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		save(t, "active", "")
		checked := stubCheck(t, nil)

		err := Provision("box", ProvisionOptions{})
		if err == nil {
			t.Fatal("an active sandbox must not be provisioned again by accident")
		}
		for _, want := range []string{"active", "--force"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should mention %q", err, want)
			}
		}
		if *checked {
			t.Error("the machine must not be touched when the status refuses the command")
		}
	})

	t.Run("--force reaches the machine", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		save(t, "active", "")
		checked := stubCheck(t, errors.New("not provisioned"))

		err := Provision("box", ProvisionOptions{Force: true})
		if err == nil || !strings.Contains(err.Error(), "not provisioned") {
			t.Fatalf("expected the marker check to stop the command, got %v", err)
		}
		if !*checked {
			t.Error("--force should pass the status gate and check the marker")
		}
	})

	t.Run("refuses to start when a required variable is missing", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		t.Setenv("GITHUB_TOKEN", "")
		save(t, "provisioning", "")
		checked := stubCheck(t, nil)

		err := Provision("box", ProvisionOptions{})
		if err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN") {
			t.Fatalf("expected the missing variable to be named, got %v", err)
		}
		if *checked {
			t.Error("a missing credential must stop the command before it touches the machine")
		}
	})

	t.Run("asks for the credentials of the recorded auth mode", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		// The record was created with a subscription, so the AWS keys
		// stubCreateEnv sets are not what it needs.
		t.Setenv(oauthTokenVar, "")
		save(t, "provisioning", string(AuthSubscription))
		stubCheck(t, nil)

		err := Provision("box", ProvisionOptions{})
		if err == nil || !strings.Contains(err.Error(), oauthTokenVar) {
			t.Fatalf("expected %s to be named, got %v", oauthTokenVar, err)
		}
	})

	t.Run("reports a sandbox that is not recorded", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)

		err := Provision("nobody", ProvisionOptions{})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}
