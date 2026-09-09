package cli

import (
	"strings"
	"testing"
)

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
	if result.Count("spec") != 1 || result.Count("dryRun") != 1 || result.Err != nil {
		t.Fatalf("invalid alias counts or error: %+v", result)
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

func TestParseFlags_ReportsUnknownArgumentsAndCollectsKnownFlags(t *testing.T) {
	flags := FlagSet{
		"once": {Names: []string{"--once"}, Type: BoolFlag},
	}

	result := ParseFlags([]string{"--unknown", "--once", "--also-unknown", "value"}, flags)

	if !result.Bool("once") {
		t.Error("expected once=true despite unknown flags")
	}
	if result.Err == nil || result.Err.Error() != `unknown argument "--unknown"` {
		t.Fatalf("expected the first unknown argument, got %v", result.Err)
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

func TestParseFlags_MissingValueSurvivesLaterAlias(t *testing.T) {
	flags := FlagSet{
		"spec": {Names: []string{"--spec", "-s"}, Type: StringFlag},
	}

	result := ParseFlags([]string{"--spec", "-s", "auth"}, flags)
	if result.Count("spec") != 2 || result.String("spec") != "auth" || result.Err == nil || result.Err.Error() != "--spec requires a value" {
		t.Fatalf("missing value was lost across aliases: %+v", result)
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

// Strings holds one value per flag, so a repeated string flag would drop a
// value the caller typed. The parser refuses it, and refuses it once, rather
// than leaving each command to answer the same mistake its own way. A repeated
// boolean flag drops nothing, so it stays valid.
func TestParseFlags_RejectsARepeatedStringFlag(t *testing.T) {
	set := FlagSet{
		"status": {Names: []string{"--status"}, Type: StringFlag},
		"spec":   {Names: []string{"--spec", "-s"}, Type: StringFlag},
		"json":   {Names: []string{"--json"}, Type: BoolFlag},
	}

	repeated := ParseFlags([]string{"--status", "draft", "--status", "done"}, set)
	if repeated.Err == nil || !strings.Contains(repeated.Err.Error(), "--status must be supplied only once") {
		t.Errorf("repeated --status: Err = %v, want a refusal", repeated.Err)
	}

	// An alias is the same flag, so the pair is still a repeat.
	aliased := ParseFlags([]string{"--spec", "one", "-s", "two"}, set)
	if aliased.Err == nil {
		t.Error("a flag repeated under its alias must be refused")
	}

	// A repeated boolean discards nothing, and one value per flag is fine.
	fine := ParseFlags([]string{"--json", "--json", "--status", "done"}, set)
	if fine.Err != nil {
		t.Errorf("Err = %v, want nil", fine.Err)
	}
	if got := fine.String("status"); got != "done" {
		t.Errorf("status = %q, want done", got)
	}
}
