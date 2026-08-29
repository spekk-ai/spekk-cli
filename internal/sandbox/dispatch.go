package sandbox

import (
	"fmt"
	"strings"
)

// ValidProviders lists the provider names accepted by --provider.
var ValidProviders = []string{"digitalocean", "manual"}

// DOOnlyFlags are flags that only make sense with --provider digitalocean.
var DOOnlyFlags = []string{"--region", "--size", "--vpc", "--project"}

// ManualOnlyFlags are flags that only make sense with --provider manual.
var ManualOnlyFlags = []string{"--ip", "--ssh-key"}

// ResolveProviderName infers the provider name from the --provider flag value
// and whether --ip was supplied. Returns the resolved provider name or an error.
//
// Rules:
//   - Explicit --provider value is used as-is (validated against ValidProviders)
//   - If --provider is omitted and --ip is set, defaults to "manual"
//   - If --provider is omitted and --ip is not set, defaults to "digitalocean"
func ResolveProviderName(providerFlag string, ipSet bool) (string, error) {
	if providerFlag != "" {
		for _, v := range ValidProviders {
			if providerFlag == v {
				return providerFlag, nil
			}
		}
		return "", fmt.Errorf("invalid provider %q: valid values are %s", providerFlag, strings.Join(ValidProviders, ", "))
	}
	if ipSet {
		return "manual", nil
	}
	return "digitalocean", nil
}

// ValidateProviderFlags checks that no provider-incompatible flags are set.
// setFlags maps flag names (e.g. "--region") to whether they were provided.
func ValidateProviderFlags(provider string, setFlags map[string]bool) error {
	switch provider {
	case "manual":
		var bad []string
		for _, f := range DOOnlyFlags {
			if setFlags[f] {
				bad = append(bad, f)
			}
		}
		if len(bad) > 0 {
			return fmt.Errorf("flags %s cannot be used with --provider manual", strings.Join(bad, ", "))
		}
	case "digitalocean":
		var bad []string
		for _, f := range ManualOnlyFlags {
			if setFlags[f] {
				bad = append(bad, f)
			}
		}
		if len(bad) > 0 {
			return fmt.Errorf("flags %s cannot be used with --provider digitalocean", strings.Join(bad, ", "))
		}
	}
	return nil
}

// ProviderByName returns a Provider implementation for the given name.
// For "digitalocean" it creates a DOProvider (requires API token env var).
// For "manual" it returns a ManualProvider.
func ProviderByName(name string) (Provider, error) {
	switch name {
	case "digitalocean":
		return NewDOProvider()
	case "manual":
		return &ManualProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// ProviderFromMeta instantiates the correct Provider based on stored sandbox metadata.
func ProviderFromMeta(meta *SandboxMeta) (Provider, error) {
	if meta.Provider == "" {
		// Legacy sandbox without provider field — assume digitalocean.
		return NewDOProvider()
	}
	return ProviderByName(meta.Provider)
}
