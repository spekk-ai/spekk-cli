package install

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ResolveSourceSkill validates rawURL and returns the skill name to use for
// the destination filename when installing via --source.
//
// If skillArg is non-empty, it wins outright — the URL's basename is ignored,
// matching the assertion's rule that `--source` install destinations use the
// positional `<skill>` arg. The arg is still validated as a single path
// segment (see validateSkillName).
//
// If skillArg is empty, the name is derived from the URL's path basename with
// any `.md` suffix stripped. Note that path.Base strips a trailing slash, so
// `https://x.com/a/` derives `a` rather than erroring. A genuinely empty or
// unusable basename (e.g. the URL path is just `/`), or one that isn't a plain
// path segment (e.g. `..`), yields an error asking the caller to pass
// `<skill>` explicitly.
//
// In every case, rawURL must parse and have an http(s) scheme and a host;
// otherwise this function returns a descriptive error and skillArg is not
// returned.
func ResolveSourceSkill(rawURL, skillArg string) (string, error) {
	if err := validateSourceURL(rawURL); err != nil {
		return "", err
	}
	if skillArg != "" {
		if err := validateSkillName(skillArg); err != nil {
			return "", err
		}
		return skillArg, nil
	}

	u, _ := url.Parse(rawURL) // already validated above
	base := path.Base(u.Path)
	if base == "." || base == "/" || base == "" {
		return "", fmt.Errorf("cannot derive skill name from URL %q: pass an explicit <skill> argument", rawURL)
	}
	derived := strings.TrimSuffix(base, ".md")
	if derived == "" {
		return "", fmt.Errorf("cannot derive skill name from URL %q: pass an explicit <skill> argument", rawURL)
	}
	// A basename like ".." survives path.Base; reject it (and anything else
	// that isn't a plain segment) so the derived name can't escape the dir.
	if err := validateSkillName(derived); err != nil {
		return "", fmt.Errorf("cannot derive skill name from URL %q: %w (pass an explicit <skill> argument)", rawURL, err)
	}
	return derived, nil
}

// validateSourceURL enforces the --source URL contract: must parse, must use
// http or https scheme, must have a host. Any other shape is rejected.
func validateSourceURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("source URL is empty")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid source URL %q: %w", rawURL, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("source URL %q must use http or https scheme", rawURL)
	}
	if u.Host == "" {
		return fmt.Errorf("source URL %q has no host", rawURL)
	}
	return nil
}
