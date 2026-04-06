package cli

import (
	"testing"
)

// helper: build the sample flag set used in the Node.js tests.
func sampleFlags() *FlagSet {
	return NewFlagSet().
		Bool("verbose", "--verbose", "-v").
		String("output", "--output", "-o").
		Bool("force", "--force")
}

// --- Defaults ---

func TestDefaults_BoolFalse(t *testing.T) {
	pf := sampleFlags().Parse([]string{})
	if pf.GetBool("verbose") != false {
		t.Fatal("expected verbose default false")
	}
	if pf.GetBool("force") != false {
		t.Fatal("expected force default false")
	}
}

func TestDefaults_StringEmpty(t *testing.T) {
	pf := sampleFlags().Parse([]string{})
	if pf.GetString("output") != "" {
		t.Fatalf("expected output default empty, got %q", pf.GetString("output"))
	}
}

// --- Boolean flags ---

func TestBooleanFlag_Long(t *testing.T) {
	pf := sampleFlags().Parse([]string{"--verbose"})
	if !pf.GetBool("verbose") {
		t.Fatal("expected verbose true")
	}
	if pf.GetBool("force") {
		t.Fatal("expected force still false")
	}
}

func TestBooleanFlag_Short(t *testing.T) {
	pf := sampleFlags().Parse([]string{"-v"})
	if !pf.GetBool("verbose") {
		t.Fatal("expected verbose true via short alias")
	}
}

// --- String flags ---

func TestStringFlag_Long(t *testing.T) {
	pf := sampleFlags().Parse([]string{"--output", "file.txt"})
	if pf.GetString("output") != "file.txt" {
		t.Fatalf("expected output 'file.txt', got %q", pf.GetString("output"))
	}
}

func TestStringFlag_Short(t *testing.T) {
	pf := sampleFlags().Parse([]string{"-o", "file.txt"})
	if pf.GetString("output") != "file.txt" {
		t.Fatalf("expected output 'file.txt', got %q", pf.GetString("output"))
	}
}

// --- Multiple flags together ---

func TestMultipleFlags(t *testing.T) {
	pf := sampleFlags().Parse([]string{"--verbose", "--force", "--output", "out.json"})
	if !pf.GetBool("verbose") {
		t.Fatal("expected verbose true")
	}
	if !pf.GetBool("force") {
		t.Fatal("expected force true")
	}
	if pf.GetString("output") != "out.json" {
		t.Fatalf("expected output 'out.json', got %q", pf.GetString("output"))
	}
}

// --- Unknown flags are ignored ---

func TestUnknownFlagsIgnored(t *testing.T) {
	pf := sampleFlags().Parse([]string{"--unknown", "--verbose"})
	if !pf.GetBool("verbose") {
		t.Fatal("expected verbose true even with unknown flag present")
	}
	// GetBool for an unknown name returns false (zero value).
	if pf.GetBool("unknown") {
		t.Fatal("unknown flag should not appear in results")
	}
}

// --- Unregistered name returns zero value ---

func TestGetBool_Unregistered(t *testing.T) {
	pf := sampleFlags().Parse([]string{})
	if pf.GetBool("nonexistent") != false {
		t.Fatal("unregistered bool should return false")
	}
}

func TestGetString_Unregistered(t *testing.T) {
	pf := sampleFlags().Parse([]string{})
	if pf.GetString("nonexistent") != "" {
		t.Fatal("unregistered string should return empty")
	}
}

// --- String flag at end of args (missing value) ---

func TestStringFlag_MissingValue(t *testing.T) {
	pf := sampleFlags().Parse([]string{"--output"})
	// No value follows --output; should stay at default "".
	if pf.GetString("output") != "" {
		t.Fatalf("expected output empty when value missing, got %q", pf.GetString("output"))
	}
}

// --- Builder-style flag set (matches Node.js builder CLI) ---

func builderFlags() *FlagSet {
	return NewFlagSet().
		Bool("once", "--once").
		Bool("dry-run", "--dry-run", "-d").
		Bool("confirm", "--confirm", "-c").
		Bool("interactive", "--interactive", "-i").
		String("spec", "--spec", "-s").
		String("assertion", "--assertion").
		Bool("help", "--help", "-h")
}

func TestBuilderFlags_Once(t *testing.T) {
	pf := builderFlags().Parse([]string{"--once"})
	if !pf.GetBool("once") {
		t.Fatal("expected once true")
	}
}

func TestBuilderFlags_DryRunShort(t *testing.T) {
	pf := builderFlags().Parse([]string{"-d"})
	if !pf.GetBool("dry-run") {
		t.Fatal("expected dry-run true via -d")
	}
}

func TestBuilderFlags_SpecShort(t *testing.T) {
	pf := builderFlags().Parse([]string{"-s", "auth"})
	if pf.GetString("spec") != "auth" {
		t.Fatalf("expected spec 'auth', got %q", pf.GetString("spec"))
	}
}

func TestBuilderFlags_ConfirmShort(t *testing.T) {
	pf := builderFlags().Parse([]string{"-c"})
	if !pf.GetBool("confirm") {
		t.Fatal("expected confirm true via -c")
	}
}

func TestBuilderFlags_Combo(t *testing.T) {
	pf := builderFlags().Parse([]string{"--once", "--spec", "auth", "--confirm"})
	if !pf.GetBool("once") {
		t.Fatal("expected once")
	}
	if pf.GetString("spec") != "auth" {
		t.Fatalf("expected spec 'auth', got %q", pf.GetString("spec"))
	}
	if !pf.GetBool("confirm") {
		t.Fatal("expected confirm")
	}
	if pf.GetBool("dry-run") {
		t.Fatal("expected dry-run false")
	}
}

// --- Parser-style flag set (matches Node.js parser CLI) ---

func parserFlags() *FlagSet {
	return NewFlagSet().
		Bool("all", "--all").
		Bool("all-branches", "--all-branches").
		String("spec", "--spec", "-s").
		String("assertion", "--assertion")
}

func TestParserFlags_All(t *testing.T) {
	pf := parserFlags().Parse([]string{"--all"})
	if !pf.GetBool("all") {
		t.Fatal("expected all true")
	}
}

func TestParserFlags_SpecWithShort(t *testing.T) {
	pf := parserFlags().Parse([]string{"-s", "auth"})
	if pf.GetString("spec") != "auth" {
		t.Fatalf("expected spec 'auth', got %q", pf.GetString("spec"))
	}
}

func TestParserFlags_Assertion(t *testing.T) {
	pf := parserFlags().Parse([]string{"--assertion", "foo"})
	if pf.GetString("assertion") != "foo" {
		t.Fatalf("expected assertion 'foo', got %q", pf.GetString("assertion"))
	}
}

// --- Chaining API ---

func TestFlagSetChaining(t *testing.T) {
	fs := NewFlagSet().
		Bool("watch", "--watch", "-w").
		String("port", "--port", "-p").
		Bool("help", "--help", "-h")

	pf := fs.Parse([]string{"-w", "--port", "8080"})
	if !pf.GetBool("watch") {
		t.Fatal("expected watch true")
	}
	if pf.GetString("port") != "8080" {
		t.Fatalf("expected port '8080', got %q", pf.GetString("port"))
	}
	if pf.GetBool("help") {
		t.Fatal("expected help false")
	}
}

// --- Positional / non-flag arguments mixed in ---

func TestNonFlagArgsMixed(t *testing.T) {
	// Positional arguments like "subcommand" should be silently skipped
	// (they are unknown strings, not registered flags).
	pf := sampleFlags().Parse([]string{"subcommand", "--verbose", "trailing"})
	if !pf.GetBool("verbose") {
		t.Fatal("expected verbose true despite positional args")
	}
}

// --- Empty FlagSet parses without error ---

func TestEmptyFlagSet(t *testing.T) {
	fs := NewFlagSet()
	pf := fs.Parse([]string{"--anything", "value"})
	if pf.GetBool("anything") != false {
		t.Fatal("empty flag set should return defaults")
	}
}

// --- String flag value that looks like a flag ---

func TestStringFlag_ValueLooksLikeFlag(t *testing.T) {
	// --output --verbose : the value of --output is literally "--verbose"
	pf := sampleFlags().Parse([]string{"--output", "--verbose"})
	if pf.GetString("output") != "--verbose" {
		t.Fatalf("expected output '--verbose', got %q", pf.GetString("output"))
	}
	// verbose should be false because "--verbose" was consumed as output's value
	if pf.GetBool("verbose") {
		t.Fatal("expected verbose false (consumed as output value)")
	}
}

// --- Multiple aliases for same flag ---

func TestMultipleAliases(t *testing.T) {
	fs := NewFlagSet().
		Bool("verbose", "--verbose", "-v", "-V", "--debug")

	for _, arg := range []string{"--verbose", "-v", "-V", "--debug"} {
		pf := fs.Parse([]string{arg})
		if !pf.GetBool("verbose") {
			t.Fatalf("expected verbose true via %s", arg)
		}
	}
}
