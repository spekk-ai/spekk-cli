package sandbox

import (
	"strings"
	"testing"
)

func TestResolveProviderName(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		existingMachine bool
		want            string
		wantErr         bool
	}{
		{"explicit digitalocean", "digitalocean", false, "digitalocean", false},
		{"explicit none", ProviderNone, false, ProviderNone, false},
		{"explicit none naming a machine", ProviderNone, true, ProviderNone, false},
		{"invalid provider", "aws", false, "", true},
		// Naming a machine is what says there is nothing to create, and
		// --ssh-key names one as surely as --ip does.
		{"a named machine means none", "", true, ProviderNone, false},
		{"no named machine means digitalocean", "", false, "digitalocean", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProviderName(tt.provider, tt.existingMachine)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Asking for a new machine and naming an existing one at the same time is a
// contradiction. Ignoring either half bills a droplet nobody wanted, or
// ignores the machine the operator named.
func TestValidateProviderFlags(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		set      []string
		wantErr  string
	}{
		{"cloud flags with a machine you already have", ProviderNone, []string{"--region"}, "--region"},
		{"machine flags with digitalocean", "digitalocean", []string{"--ip", "--ssh-key"}, "--ip, --ssh-key"},
		{"cloud flags with digitalocean", "digitalocean", []string{"--region", "--size"}, ""},
		{"machine flags with none", ProviderNone, []string{"--ip", "--ssh-key"}, ""},
		{"ssh-user with none", ProviderNone, []string{"--ip", "--ssh-key", "--ssh-user"}, ""},
		{"ssh-user with digitalocean", "digitalocean", []string{"--ssh-user"}, "--ssh-user"},
		{"nothing set", "digitalocean", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := map[string]bool{}
			for _, f := range tt.set {
				set[f] = true
			}
			err := ValidateProviderFlags(tt.provider, set)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error naming %s", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), tt.provider) {
				t.Errorf("error %q should name both %s and the provider", err, tt.wantErr)
			}
		})
	}
}

// A machine no cloud owns has no provider to build.
func TestProviderByNameNoneIsNil(t *testing.T) {
	p, err := ProviderByName(ProviderNone)
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Errorf("want a nil Provider, got %T", p)
	}
	if p, err := ProviderFromMeta(&SandboxMeta{Provider: ProviderNone}); err != nil || p != nil {
		t.Errorf("ProviderFromMeta = %v, %v; want nil, nil", p, err)
	}
}
