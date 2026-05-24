package install

import (
	"fmt"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// SkillsUsageText is the help text printed for `spekk skills` when no
// subcommand is supplied or when `--help` is passed.
const SkillsUsageText = `spekk skills - Inspect skills available to an agent

USAGE:
  spekk skills <subcommand>

SUBCOMMANDS:
  list <agent>   List every skill available to <agent> across all scopes
                 (local + global + embedded), showing each entry's source

ARGUMENTS:
  <agent>   One of: coach, builder, observer

EXAMPLES:
  spekk skills list coach
`

// FormatSkillsList renders the skills available to an agent. Each row shows
// the skill name and its source directory (or "(embedded)"). When the list
// is empty, returns a "no skills found" message so callers can print it
// unconditionally.
func FormatSkillsList(agent string, skills []cli.SkillEntry) string {
	var b strings.Builder
	if len(skills) == 0 {
		fmt.Fprintf(&b, "No skills found for agent %q.\n", agent)
		return b.String()
	}
	fmt.Fprintf(&b, "Skills for %s:\n", agent)
	for _, s := range skills {
		fmt.Fprintf(&b, "  %s  —  %s\n", s.Name, s.Source)
	}
	return b.String()
}

// ValidateSkillsAgent rejects any agent name not in ValidAgents using the
// same error shape as `spekk install`, so the two surfaces stay consistent.
func ValidateSkillsAgent(agent string) error {
	if !isValidAgent(agent) {
		return unknownAgentError(agent)
	}
	return nil
}
