package agent

import (
	"strings"
	"testing"
)

// The default harness must resolve to claude-code whether the name is empty
// (no --harness, no SPEKK_HARNESS) or given via the "claude" alias.
func TestResolveProfile_DefaultAndAlias(t *testing.T) {
	for _, name := range []string{"", "claude", "claude-code"} {
		p, err := ResolveProfile(name)
		if err != nil {
			t.Fatalf("ResolveProfile(%q) errored: %v", name, err)
		}
		if p.Name != "claude-code" || p.Binary != "claude" {
			t.Fatalf("ResolveProfile(%q) = %s/%s, want claude-code/claude", name, p.Name, p.Binary)
		}
	}
}

// An unknown harness must fail fast rather than return a zero profile a caller
// might spawn.
func TestResolveProfile_UnknownFailsFast(t *testing.T) {
	if _, err := ResolveProfile("nope"); err == nil {
		t.Fatal("ResolveProfile(\"nope\") should error")
	}
}

// The opencode alias resolves to the opencode profile (not claude-code), so a
// user can select opencode by name in both --harness and SPEKK_HARNESS.
func TestResolveProfile_Opencode(t *testing.T) {
	p, err := ResolveProfile("opencode")
	if err != nil {
		t.Fatalf("ResolveProfile(\"opencode\") errored: %v", err)
	}
	if p.Name != "opencode" || p.Binary != "opencode" {
		t.Fatalf("ResolveProfile(\"opencode\") = %s/%s, want opencode/opencode", p.Name, p.Binary)
	}
}

// The aider alias resolves to the aider profile (not claude-code), so a user
// can select aider by name in both --harness and SPEKK_HARNESS.
func TestResolveProfile_Aider(t *testing.T) {
	p, err := ResolveProfile("aider")
	if err != nil {
		t.Fatalf("ResolveProfile(\"aider\") errored: %v", err)
	}
	if p.Name != "aider" || p.Binary != "aider" {
		t.Fatalf("ResolveProfile(\"aider\") = %s/%s, want aider/aider", p.Name, p.Binary)
	}
}

// The aider profile's resolved argv must follow aider's own CLI conventions in
// every mode. Interactive and the interactive builder launch bare aider (no
// flags) so it opens a chat session and waits for input; headless uses
// `--yes-always --message` so aider auto-confirms and processes the single
// message before exiting.
func TestAiderProfile_ResolvedArgv(t *testing.T) {
	p, err := ResolveProfile("aider")
	if err != nil {
		t.Fatalf("ResolveProfile(\"aider\") errored: %v", err)
	}
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{msg}},
		{"headless", p.HeadlessArgs(msg), []string{"--yes-always", "--message", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// The aider profile's not-found guidance must name aider and point at aider's
// install instructions — never Claude's.
func TestAiderProfile_NotFoundNamesAider(t *testing.T) {
	p, err := ResolveProfile("aider")
	if err != nil {
		t.Fatalf("ResolveProfile(\"aider\") errored: %v", err)
	}
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Aider") {
		t.Errorf("first not-found line does not name aider: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) || !strings.Contains(l2, "aider.chat") {
		t.Errorf("second not-found line does not point at aider's install URL: %q", l2)
	}
	if strings.Contains(l1+l2, "Claude") || strings.Contains(l1+l2, "claude.ai") {
		t.Errorf("aider guidance mentions Claude: %q / %q", l1, l2)
	}
}

// The hermes name resolves to the hermes profile (not claude-code), so a user
// can select hermes by name in both --harness and SPEKK_HARNESS.
func TestResolveProfile_Hermes(t *testing.T) {
	p, err := ResolveProfile("hermes")
	if err != nil {
		t.Fatalf("ResolveProfile(\"hermes\") errored: %v", err)
	}
	if p.Name != "hermes" || p.Binary != "hermes" {
		t.Fatalf("ResolveProfile(\"hermes\") = %s/%s, want hermes/hermes", p.Name, p.Binary)
	}
}

// The hermes profile's resolved argv must follow hermes's own CLI conventions —
// not a copy of the claude flags. Interactive and the interactive builder use
// `--tui -z` so hermes seeds the prompt and stays interactive; headless uses
// `--yolo --cli -z` so hermes runs the single prompt non-interactively and
// bypasses approval prompts. The permission-skip flag is hermes's real `--yolo`
// and appears only in headless.
func TestHermesProfile_ResolvedArgv(t *testing.T) {
	p, err := ResolveProfile("hermes")
	if err != nil {
		t.Fatalf("ResolveProfile(\"hermes\") errored: %v", err)
	}
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{"--tui", "-z", msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{"--tui", "-z", msg}},
		{"headless", p.HeadlessArgs(msg), []string{"--yolo", "--cli", "-z", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	// The permission-skip flag must be hermes's real --yolo, never a claude or
	// opencode flag copied across.
	head := strings.Join(p.HeadlessArgs(msg), " ")
	if !strings.Contains(head, "--yolo") {
		t.Errorf("headless argv missing hermes's --yolo: %q", head)
	}
	for _, foreign := range []string{"--dangerously-skip-permissions", "--auto", "--yes-always"} {
		if strings.Contains(head, foreign) {
			t.Errorf("hermes headless argv copied a foreign permission flag %q: %q", foreign, head)
		}
	}
}

// The hermes profile's not-found guidance must name Hermes and point at
// Hermes's install docs — never Claude's.
func TestHermesProfile_NotFoundNamesHermes(t *testing.T) {
	p, err := ResolveProfile("hermes")
	if err != nil {
		t.Fatalf("ResolveProfile(\"hermes\") errored: %v", err)
	}
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Hermes") {
		t.Errorf("first not-found line does not name Hermes: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) || !strings.Contains(l2, "hermes-agent.nousresearch.com") {
		t.Errorf("second not-found line does not point at Hermes's install docs: %q", l2)
	}
	if strings.Contains(l1+l2, "Claude") || strings.Contains(l1+l2, "claude.ai") {
		t.Errorf("hermes guidance mentions Claude: %q / %q", l1, l2)
	}
}

// The codex name resolves to the codex profile (not claude-code), so a user
// can select codex by name in both --harness and SPEKK_HARNESS.
func TestResolveProfile_Codex(t *testing.T) {
	p, err := ResolveProfile("codex")
	if err != nil {
		t.Fatalf("ResolveProfile(\"codex\") errored: %v", err)
	}
	if p.Name != "codex" || p.Binary != "codex" {
		t.Fatalf("ResolveProfile(\"codex\") = %s/%s, want codex/codex", p.Name, p.Binary)
	}
}

// The codex profile's resolved argv must follow codex's own CLI conventions —
// not a copy of the claude flags. Interactive and the interactive builder launch
// bare codex (the prompt is the session's initial-prompt positional and codex
// stays interactive); headless uses `exec --dangerously-bypass-approvals-and-sandbox`
// so codex runs the single prompt non-interactively and skips approval prompts.
// The permission-skip flag is codex's real --dangerously-bypass-approvals-and-sandbox
// and appears only in headless.
func TestCodexProfile_ResolvedArgv(t *testing.T) {
	p, err := ResolveProfile("codex")
	if err != nil {
		t.Fatalf("ResolveProfile(\"codex\") errored: %v", err)
	}
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{msg}},
		{"headless", p.HeadlessArgs(msg), []string{"exec", "--dangerously-bypass-approvals-and-sandbox", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	// The permission-skip flag must be codex's real
	// --dangerously-bypass-approvals-and-sandbox, never a claude/opencode/aider/
	// hermes flag copied across, and must be absent interactively (a human is
	// present to approve there).
	head := strings.Join(p.HeadlessArgs(msg), " ")
	if !strings.Contains(head, "--dangerously-bypass-approvals-and-sandbox") {
		t.Errorf("headless argv missing codex's --dangerously-bypass-approvals-and-sandbox: %q", head)
	}
	for _, foreign := range []string{"--dangerously-skip-permissions", "--auto", "--yes-always", "--yolo"} {
		if strings.Contains(head, foreign) {
			t.Errorf("codex headless argv copied a foreign permission flag %q: %q", foreign, head)
		}
	}
	if inter := strings.Join(p.InteractiveArgs(msg), " "); strings.Contains(inter, "--dangerously-bypass-approvals-and-sandbox") {
		t.Errorf("codex interactive argv must not carry the permission-skip flag: %q", inter)
	}
}

// The codex profile's not-found guidance must name Codex and point at codex's
// install docs — never Claude's.
func TestCodexProfile_NotFoundNamesCodex(t *testing.T) {
	p, err := ResolveProfile("codex")
	if err != nil {
		t.Fatalf("ResolveProfile(\"codex\") errored: %v", err)
	}
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Codex") {
		t.Errorf("first not-found line does not name Codex: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) || !strings.Contains(l2, "github.com/openai/codex") {
		t.Errorf("second not-found line does not point at codex's install docs: %q", l2)
	}
	if strings.Contains(l1+l2, "Claude") || strings.Contains(l1+l2, "claude.ai") {
		t.Errorf("codex guidance mentions Claude: %q / %q", l1, l2)
	}
}

// The gemini name resolves to the gemini profile (not claude-code), so a user
// can select gemini by name in both --harness and SPEKK_HARNESS.
func TestResolveProfile_Gemini(t *testing.T) {
	p, err := ResolveProfile("gemini")
	if err != nil {
		t.Fatalf("ResolveProfile(\"gemini\") errored: %v", err)
	}
	if p.Name != "gemini" || p.Binary != "gemini" {
		t.Fatalf("ResolveProfile(\"gemini\") = %s/%s, want gemini/gemini", p.Name, p.Binary)
	}
}

// The gemini profile's resolved argv must follow gemini's own CLI conventions —
// not a copy of the claude flags. Interactive and the interactive builder launch
// `gemini -i <prompt>` (`-i/--prompt-interactive` seeds the prompt and stays
// interactive); headless uses `--yolo -p <prompt>` so gemini runs the single
// prompt non-interactively (`-p/--prompt`) and skips approval prompts
// (`-y/--yolo`). The permission-skip flag is gemini's real --yolo and appears
// only in headless.
func TestGeminiProfile_ResolvedArgv(t *testing.T) {
	p, err := ResolveProfile("gemini")
	if err != nil {
		t.Fatalf("ResolveProfile(\"gemini\") errored: %v", err)
	}
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{"-i", msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{"-i", msg}},
		{"headless", p.HeadlessArgs(msg), []string{"--yolo", "-p", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	// The permission-skip flag must be gemini's real --yolo, never a
	// claude/opencode/aider/codex flag copied across, and must be absent
	// interactively (a human is present to approve there). --yolo is gemini's
	// own flag, so it is not in the foreign list.
	head := strings.Join(p.HeadlessArgs(msg), " ")
	if !strings.Contains(head, "--yolo") {
		t.Errorf("headless argv missing gemini's --yolo: %q", head)
	}
	for _, foreign := range []string{"--dangerously-skip-permissions", "--auto", "--yes-always", "--dangerously-bypass-approvals-and-sandbox"} {
		if strings.Contains(head, foreign) {
			t.Errorf("gemini headless argv copied a foreign permission flag %q: %q", foreign, head)
		}
	}
	if inter := strings.Join(p.InteractiveArgs(msg), " "); strings.Contains(inter, "--yolo") {
		t.Errorf("gemini interactive argv must not carry the permission-skip flag: %q", inter)
	}
}

// The gemini profile's not-found guidance must name the Gemini CLI and point at
// its install docs — never Claude's.
func TestGeminiProfile_NotFoundNamesGemini(t *testing.T) {
	p, err := ResolveProfile("gemini")
	if err != nil {
		t.Fatalf("ResolveProfile(\"gemini\") errored: %v", err)
	}
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Gemini CLI") {
		t.Errorf("first not-found line does not name the Gemini CLI: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) || !strings.Contains(l2, "github.com/google-gemini/gemini-cli") {
		t.Errorf("second not-found line does not point at gemini's install docs: %q", l2)
	}
	if strings.Contains(l1+l2, "Claude") || strings.Contains(l1+l2, "claude.ai") {
		t.Errorf("gemini guidance mentions Claude: %q / %q", l1, l2)
	}
}

// Harness selection follows the precedence --harness flag > SPEKK_HARNESS env >
// default claude-code, and an explicit flag overrides a conflicting env var.
func TestResolveHarness_Precedence(t *testing.T) {
	t.Setenv(HarnessEnvVar, "") // start from a clean env for the default case

	// 1. Default: no flag, no env -> claude-code.
	if p, err := ResolveHarness(""); err != nil || p.Name != "claude-code" {
		t.Fatalf("default: got %s/%v, want claude-code/nil", p.Name, err)
	}

	// 2. Env only: SPEKK_HARNESS selects the harness when no flag is given.
	t.Setenv(HarnessEnvVar, "opencode")
	if p, err := ResolveHarness(""); err != nil || p.Name != "opencode" {
		t.Fatalf("env: got %s/%v, want opencode/nil", p.Name, err)
	}

	// 3. Flag overrides env: --harness=claude wins over SPEKK_HARNESS=opencode.
	if p, err := ResolveHarness("claude"); err != nil || p.Name != "claude-code" {
		t.Fatalf("flag-over-env: got %s/%v, want claude-code/nil", p.Name, err)
	}
}

// An unknown harness must fail fast with the identical error whether the name
// came from the flag or the env var — neither may reach a spawn.
func TestResolveHarness_UnknownIdenticalError(t *testing.T) {
	fromFlag, flagErr := ResolveHarness("bogus")
	if flagErr == nil {
		t.Fatal("unknown flag harness should error")
	}

	t.Setenv(HarnessEnvVar, "bogus")
	fromEnv, envErr := ResolveHarness("")
	if envErr == nil {
		t.Fatal("unknown env harness should error")
	}

	if flagErr.Error() != envErr.Error() {
		t.Fatalf("errors differ by source:\n flag: %q\n env:  %q", flagErr.Error(), envErr.Error())
	}
	if !strings.Contains(flagErr.Error(), "opencode") || !strings.Contains(flagErr.Error(), "claude-code") {
		t.Fatalf("error should list valid harness names, got %q", flagErr.Error())
	}
	if fromFlag.Binary != "" || fromEnv.Binary != "" {
		t.Fatal("a failed resolution must return the zero profile, not a spawnable one")
	}
}

// The argv the default profile produces at each launch site must be
// byte-for-byte what the launch sites hardcoded before harnesses were
// selectable: interactive, the interactive-builder system prompt, and headless.
func TestDefaultProfile_ResolvedArgv(t *testing.T) {
	p := DefaultProfile()
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{"--dangerously-skip-permissions", msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{"--dangerously-skip-permissions", "--system-prompt", msg}},
		{"headless", p.HeadlessArgs(msg), []string{"-p", "--dangerously-skip-permissions", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// The opencode profile's resolved argv must follow opencode's own CLI
// conventions in every mode — not a copy of the claude flags. Interactive and
// the interactive builder route through `run -i` and carry the prompt as a bare
// positional message (a bare positional on the top-level `opencode` command is
// read as the project dir, not a prompt, so the message would be dropped);
// headless uses `run --auto` with the message as a bare positional. The bare
// `opencode` command receives no flags at all.
func TestOpencodeProfile_ResolvedArgv(t *testing.T) {
	p, err := ResolveProfile("opencode")
	if err != nil {
		t.Fatalf("ResolveProfile(\"opencode\") errored: %v", err)
	}
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{"run", "-i", msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{"run", "-i", msg}},
		{"headless", p.HeadlessArgs(msg), []string{"run", "--auto", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// The opencode profile's not-found guidance must name opencode and point at
// opencode's install instructions — never Claude's.
func TestOpencodeProfile_NotFoundNamesOpencode(t *testing.T) {
	p, err := ResolveProfile("opencode")
	if err != nil {
		t.Fatalf("ResolveProfile(\"opencode\") errored: %v", err)
	}
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "opencode") {
		t.Errorf("first not-found line does not name opencode: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) || !strings.Contains(l2, "opencode.ai") {
		t.Errorf("second not-found line does not point at opencode's install URL: %q", l2)
	}
	if strings.Contains(l1+l2, "Claude") || strings.Contains(l1+l2, "claude.ai") {
		t.Errorf("opencode guidance mentions Claude: %q / %q", l1, l2)
	}
}

// The not-found guidance must come from the profile and name that harness, so a
// non-claude harness never tells the user to install Claude Code.
func TestProfile_NotFoundNamesHarness(t *testing.T) {
	p := DefaultProfile()
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Claude Code") {
		t.Errorf("first not-found line does not name the harness: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) {
		t.Errorf("second not-found line does not carry the install URL: %q", l2)
	}

	other := Profile{DisplayName: "Widget", InstallURL: "https://widget.example"}
	o1, o2 := other.notFoundLines()
	if strings.Contains(o1+o2, "Claude") {
		t.Errorf("non-claude harness guidance mentions Claude: %q / %q", o1, o2)
	}
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
