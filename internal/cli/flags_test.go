package cli

import "testing"

func TestParseFlags_BooleanFlags(t *testing.T) {
	flags := FlagSet{
		"once":    {Names: []string{"--once"}, Type: BoolFlag},
		"dryRun":  {Names: []string{"--dry-run", "-d"}, Type: BoolFlag},
		"watch":   {Names: []string{"--watch", "-w"}, Type: BoolFlag},
		"confirm": {Names: []string{"--confirm", "-c"}, Type: BoolFlag},
	}

	result := ParseFlags([]string{"--once", "--dry-run"}, flags)

	if !result.Bool("once") {
		t.Error("expected once=true")
	}
	if !result.Bool("dryRun") {
		t.Error("expected dryRun=true")
	}
	if result.Bool("watch") {
		t.Error("expected watch=false")
	}
	if result.Bool("confirm") {
		t.Error("expected confirm=false")
	}
}

func TestParseFlags_StringFlags(t *testing.T) {
	flags := FlagSet{
		"spec":      {Names: []string{"--spec", "-s"}, Type: StringFlag},
		"assertion": {Names: []string{"--assertion"}, Type: StringFlag},
	}

	result := ParseFlags([]string{"--spec", "auth", "--assertion", "login-flow"}, flags)

	if got := result.String("spec"); got != "auth" {
		t.Errorf("expected spec=auth, got %q", got)
	}
	if got := result.String("assertion"); got != "login-flow" {
		t.Errorf("expected assertion=login-flow, got %q", got)
	}
}

func TestParseFlags_ShortAliases(t *testing.T) {
	flags := FlagSet{
		"spec":   {Names: []string{"--spec", "-s"}, Type: StringFlag},
		"dryRun": {Names: []string{"--dry-run", "-d"}, Type: BoolFlag},
	}

	result := ParseFlags([]string{"-s", "auth", "-d"}, flags)

	if got := result.String("spec"); got != "auth" {
		t.Errorf("expected spec=auth via -s, got %q", got)
	}
	if !result.Bool("dryRun") {
		t.Error("expected dryRun=true via -d")
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	flags := FlagSet{
		"once": {Names: []string{"--once"}, Type: BoolFlag},
		"spec": {Names: []string{"--spec"}, Type: StringFlag},
	}

	result := ParseFlags([]string{}, flags)

	if result.Bool("once") {
		t.Error("expected default once=false")
	}
	if got := result.String("spec"); got != "" {
		t.Errorf("expected default spec=\"\", got %q", got)
	}
}

func TestParseFlags_UnknownFlagsIgnored(t *testing.T) {
	flags := FlagSet{
		"once": {Names: []string{"--once"}, Type: BoolFlag},
	}

	result := ParseFlags([]string{"--unknown", "--once", "--also-unknown", "value"}, flags)

	if !result.Bool("once") {
		t.Error("expected once=true despite unknown flags")
	}
}

func TestParseFlags_MixedBoolAndString(t *testing.T) {
	flags := FlagSet{
		"all":         {Names: []string{"--all"}, Type: BoolFlag},
		"allBranches": {Names: []string{"--all-branches"}, Type: BoolFlag},
		"spec":        {Names: []string{"--spec", "-s"}, Type: StringFlag},
		"assertion":   {Names: []string{"--assertion"}, Type: StringFlag},
		"interactive": {Names: []string{"--interactive", "-i"}, Type: BoolFlag},
	}

	result := ParseFlags([]string{"--all", "-s", "my-spec", "--interactive"}, flags)

	if !result.Bool("all") {
		t.Error("expected all=true")
	}
	if result.Bool("allBranches") {
		t.Error("expected allBranches=false")
	}
	if got := result.String("spec"); got != "my-spec" {
		t.Errorf("expected spec=my-spec, got %q", got)
	}
	if got := result.String("assertion"); got != "" {
		t.Errorf("expected assertion=\"\", got %q", got)
	}
	if !result.Bool("interactive") {
		t.Error("expected interactive=true")
	}
}

func TestParseFlags_StringFlagAtEnd(t *testing.T) {
	flags := FlagSet{
		"spec": {Names: []string{"--spec"}, Type: StringFlag},
	}

	// String flag at end with no value — should not panic
	result := ParseFlags([]string{"--spec"}, flags)

	if got := result.String("spec"); got != "" {
		t.Errorf("expected spec=\"\" when no value provided, got %q", got)
	}
}

func TestParseFlags_PerCommandFlagSets(t *testing.T) {
	// Demonstrate that each command can define its own flag set
	parserFlags := FlagSet{
		"all":         {Names: []string{"--all"}, Type: BoolFlag},
		"allBranches": {Names: []string{"--all-branches"}, Type: BoolFlag},
		"spec":        {Names: []string{"--spec", "-s"}, Type: StringFlag},
	}

	coachFlags := FlagSet{
		"interactive": {Names: []string{"--interactive", "-i"}, Type: BoolFlag},
		"confirm":     {Names: []string{"--confirm", "-c"}, Type: BoolFlag},
	}

	pResult := ParseFlags([]string{"--all", "-s", "auth"}, parserFlags)
	cResult := ParseFlags([]string{"-i", "--confirm"}, coachFlags)

	if !pResult.Bool("all") || pResult.String("spec") != "auth" {
		t.Error("parser flags not parsed correctly")
	}
	if !cResult.Bool("interactive") || !cResult.Bool("confirm") {
		t.Error("coach flags not parsed correctly")
	}
}
