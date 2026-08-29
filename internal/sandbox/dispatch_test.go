package sandbox

import (
	"testing"
)

func TestResolveProviderName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		ipSet    bool
		want     string
		wantErr  bool
	}{
		{"explicit digitalocean", "digitalocean", false, "digitalocean", false},
		{"explicit manual", "manual", false, "manual", false},
		{"explicit manual with ip", "manual", true, "manual", false},
		{"invalid provider", "aws", false, "", true},
		{"omitted with ip defaults to manual", "", true, "manual", false},
		{"omitted without ip defaults to digitalocean", "", false, "digitalocean", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProviderName(tt.provider, tt.ipSet)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateProviderFlags(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		flags    map[string]bool
		wantErr  bool
		errMsg   string
	}{
		{
			"manual with no DO flags is ok",
			"manual",
			map[string]bool{"--ip": true, "--ssh-key": true},
			false, "",
		},
		{
			"manual with --region errors",
			"manual",
			map[string]bool{"--region": true},
			true, "--region",
		},
		{
			"manual with multiple DO flags errors and names them",
			"manual",
			map[string]bool{"--region": true, "--size": true},
			true, "--region",
		},
		{
			"digitalocean with no manual flags is ok",
			"digitalocean",
			map[string]bool{"--region": true, "--size": true},
			false, "",
		},
		{
			"digitalocean with --ip errors",
			"digitalocean",
			map[string]bool{"--ip": true},
			true, "--ip",
		},
		{
			"digitalocean with --ssh-key errors",
			"digitalocean",
			map[string]bool{"--ssh-key": true},
			true, "--ssh-key",
		},
		{
			"error messages name the provider",
			"digitalocean",
			map[string]bool{"--ip": true},
			true, "digitalocean",
		},
		{
			"error messages name the provider for manual",
			"manual",
			map[string]bool{"--vpc": true},
			true, "manual",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderFlags(tt.provider, tt.flags)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if msg := err.Error(); !contains(msg, tt.errMsg) {
					t.Errorf("error %q should contain %q", msg, tt.errMsg)
				}
			}
		})
	}
}

func TestProviderByName(t *testing.T) {
	// ManualProvider can be instantiated without env vars.
	p, err := ProviderByName("manual")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*ManualProvider); !ok {
		t.Error("expected *ManualProvider")
	}

	// Unknown provider returns error.
	_, err = ProviderByName("gcp")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestProviderFromMeta(t *testing.T) {
	// Empty provider field defaults to DO (which needs env var — just test manual).
	meta := &SandboxMeta{Provider: "manual"}
	p, err := ProviderFromMeta(meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*ManualProvider); !ok {
		t.Error("expected *ManualProvider")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
