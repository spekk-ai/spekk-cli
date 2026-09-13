package agent

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// tokenPresent reports whether the flag or subcommand `tok` appears as a whole
// token in help output — bounded by string start/end or a character that is
// neither a word character nor a dash. This keeps a short flag like "-p" from
// spuriously matching inside a longer "--print", so an undefined flag cannot
// pass verification just by being a substring of a defined one.
func tokenPresent(help, tok string) bool {
	re := regexp.MustCompile(`(^|[^\w-])` + regexp.QuoteMeta(tok) + `([^\w-]|$)`)
	return re.MatchString(help)
}

// splitArgv separates a profile's emitted argv into its leading subcommand path
// and the flags it emits, ignoring the prompt message itself. A token starting
// with "-" is a flag; any other non-message token is a subcommand name.
func splitArgv(argv []string, msg string) (subs, flags []string) {
	for _, tok := range argv {
		switch {
		case tok == msg:
			continue
		case strings.HasPrefix(tok, "-"):
			flags = append(flags, tok)
		default:
			subs = append(subs, tok)
		}
	}
	return subs, flags
}

// A harness profile may only emit flags and subcommands its installed binary
// actually defines — flags must be checked against the real CLI, not written
// from memory. For every profile whose binary is on PATH this runs the binary's
// --help (and any subcommand --help the profile invokes) and asserts every flag
// and subcommand the profile emits in its interactive, system-prompt, AND
// headless argv appears in the matching help output. This is the guard for the
// class of bug where a profile ships a flag the CLI never defines and the argv
// tests pass anyway because they only assert the profile's own output.
//
// A profile whose binary is absent is skipped with a notice naming the harness —
// never silently passed (so CI without the binary stays green but never reports
// the profile as verified) and never a hard failure.
func TestProfileFlagsVerifiedAgainstCLI(t *testing.T) {
	const msg = "verification message"

	for name, profile := range harnessProfiles {
		t.Run(name, func(t *testing.T) {
			binPath, err := exec.LookPath(profile.Binary)
			if err != nil {
				t.Skipf("harness %s: binary %q not on PATH — skipping CLI flag verification (not verified, not failed)",
					profile.Name, profile.Binary)
			}

			// help(subPath) runs `<binary> <subPath...> --help` once and caches
			// the combined output, keyed by the joined subcommand path.
			cache := map[string]string{}
			help := func(subPath []string) string {
				key := strings.Join(subPath, " ")
				if out, ok := cache[key]; ok {
					return out
				}
				args := append(append([]string{}, subPath...), "--help")
				out, _ := exec.Command(binPath, args...).CombinedOutput()
				cache[key] = string(out)
				return cache[key]
			}

			// Cover every mode the profile launches under, not just one: a flag
			// wrong in headless but right interactively must still be caught.
			modes := []struct {
				mode string
				argv []string
			}{
				{"interactive", profile.InteractiveArgs(msg)},
				{"system-prompt", profile.SystemPromptArgs(msg)},
				{"headless", profile.HeadlessArgs(msg)},
			}

			for _, m := range modes {
				subs, flags := splitArgv(m.argv, msg)

				// Every subcommand name must appear in its parent's help.
				for i, sub := range subs {
					if !tokenPresent(help(subs[:i]), sub) {
						t.Errorf("%s %s: subcommand %q not defined by `%s %s --help`",
							profile.Name, m.mode, sub, profile.Binary, strings.Join(subs[:i], " "))
					}
				}

				// Every flag must appear in the help of the subcommand path it
				// runs under (the bare command when there is no subcommand).
				flagHelp := help(subs)
				for _, fl := range flags {
					if !tokenPresent(flagHelp, fl) {
						t.Errorf("%s %s: flag %q not defined by `%s %s --help`",
							profile.Name, m.mode, fl, profile.Binary, strings.TrimSpace(profile.Binary+" "+strings.Join(subs, " ")))
					}
				}
			}
		})
	}
}
