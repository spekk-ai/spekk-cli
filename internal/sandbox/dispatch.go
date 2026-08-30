package sandbox

import (
	"fmt"
	"strings"
)

// ProviderNone is the SandboxMeta.Provider value for a machine no cloud
// owns: one the operator already had, which spekk registers and equips but
// never creates or destroys.
const ProviderNone = "none"

// ValidProviders lists the values accepted by --provider.
var ValidProviders = []string{"digitalocean", ProviderNone}

// existingMachineFlags name a machine that already exists. Any one of them
// says the operator is not asking for a new one.
var existingMachineFlags = []string{"--ip", "--ssh-key"}

// cloudFlags configure a machine spekk creates, so they are meaningless for
// one it does not.
var cloudFlags = []string{"--region", "--size", "--vpc", "--project"}

// ResolveProviderName decides which provider a create is for.
//
// An explicit --provider wins. Otherwise, naming an existing machine is what
// says there is nothing to create; anything else asks for a new droplet.
func ResolveProviderName(providerFlag string, existingMachine bool) (string, error) {
	if providerFlag != "" {
		for _, v := range ValidProviders {
			if providerFlag == v {
				return providerFlag, nil
			}
		}
		return "", fmt.Errorf("invalid provider %q: valid values are %s", providerFlag, strings.Join(ValidProviders, ", "))
	}
	if existingMachine {
		return ProviderNone, nil
	}
	return "digitalocean", nil
}

// ValidateProviderFlags rejects a create that asks for both a new machine and
// an existing one. Silently ignoring either half would bill a droplet the
// operator did not want, or ignore the machine they named.
func ValidateProviderFlags(provider string, setFlags map[string]bool) error {
	wrong, forProvider := cloudFlags, ProviderNone
	if provider != ProviderNone {
		wrong, forProvider = existingMachineFlags, provider
	}
	var bad []string
	for _, f := range wrong {
		if setFlags[f] {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("flags %s cannot be used with --provider %s", strings.Join(bad, ", "), forProvider)
	}
	return nil
}

// ProviderByName returns the Provider for a name, or a nil Provider when no
// cloud owns the machine.
func ProviderByName(name string) (Provider, error) {
	switch name {
	case ProviderNone:
		return nil, nil
	case "digitalocean":
		// Unpack rather than returning the call directly: a nil
		// *DOProvider returned into a Provider result is an interface
		// that is not nil.
		p, err := NewDOProvider()
		if err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}
