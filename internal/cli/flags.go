// Package cli provides shared CLI utilities for all spekk commands.
package cli

// flagKind distinguishes boolean flags from string-valued flags.
type flagKind int

const (
	flagBool   flagKind = iota
	flagString          // expects the next argument as its value
)

// flagDef describes a single flag that the FlagSet knows about.
type flagDef struct {
	name string   // canonical name (e.g. "spec")
	kind flagKind // boolean or string
}

// FlagSet holds the definitions for a set of CLI flags.
// Each command creates its own FlagSet, registers the flags it cares about,
// then calls Parse to extract values from os.Args (or a subslice).
type FlagSet struct {
	defs   []flagDef         // ordered list so we can derive defaults
	lookup map[string]string // flag string (e.g. "--spec", "-s") → canonical name
	kinds  map[string]flagKind // canonical name → kind
}

// NewFlagSet returns an empty FlagSet ready for flag registration.
func NewFlagSet() *FlagSet {
	return &FlagSet{
		lookup: make(map[string]string),
		kinds:  make(map[string]flagKind),
	}
}

// Bool registers a boolean flag.
// The first argument is the canonical name used with GetBool.
// Remaining arguments are the flag strings that activate it
// (e.g. "--dry-run", "-d").
func (fs *FlagSet) Bool(name string, aliases ...string) *FlagSet {
	fs.defs = append(fs.defs, flagDef{name: name, kind: flagBool})
	fs.kinds[name] = flagBool
	for _, a := range aliases {
		fs.lookup[a] = name
	}
	return fs
}

// String registers a string-valued flag.
// The first argument is the canonical name used with GetString.
// Remaining arguments are the flag strings that activate it
// (e.g. "--spec", "-s").
func (fs *FlagSet) String(name string, aliases ...string) *FlagSet {
	fs.defs = append(fs.defs, flagDef{name: name, kind: flagString})
	fs.kinds[name] = flagString
	for _, a := range aliases {
		fs.lookup[a] = name
	}
	return fs
}

// ParsedFlags holds the result of parsing a set of CLI arguments.
type ParsedFlags struct {
	bools   map[string]bool
	strings map[string]string
}

// GetBool returns the value of a boolean flag (false if not set).
func (pf ParsedFlags) GetBool(name string) bool {
	return pf.bools[name]
}

// GetString returns the value of a string flag ("" if not set).
func (pf ParsedFlags) GetString(name string) string {
	return pf.strings[name]
}

// Parse processes the argument list against the registered flags.
// Unknown flags are silently ignored (not fatal).
// For string flags, the value is the next argument in the list.
func (fs *FlagSet) Parse(args []string) ParsedFlags {
	pf := ParsedFlags{
		bools:   make(map[string]bool),
		strings: make(map[string]string),
	}

	// Apply defaults: false for bools, "" for strings.
	for _, d := range fs.defs {
		switch d.kind {
		case flagBool:
			pf.bools[d.name] = false
		case flagString:
			pf.strings[d.name] = ""
		}
	}

	// Walk the argument list.
	for i := 0; i < len(args); i++ {
		name, ok := fs.lookup[args[i]]
		if !ok {
			continue // unknown flag — ignore
		}

		switch fs.kinds[name] {
		case flagBool:
			pf.bools[name] = true
		case flagString:
			i++
			if i < len(args) {
				pf.strings[name] = args[i]
			}
			// If there is no next arg, the string stays at its default ("").
		}
	}

	return pf
}
