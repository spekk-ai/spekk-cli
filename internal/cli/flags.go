// Package cli provides shared utilities for CLI command parsing.
package cli

import (
	"fmt"
	"strings"
)

// FlagType represents the type of a CLI flag.
type FlagType int

const (
	// BoolFlag is a boolean flag (presence = true).
	BoolFlag FlagType = iota
	// StringFlag is a flag that takes a string value.
	StringFlag
)

// FlagDef defines a single CLI flag with its names and type.
type FlagDef struct {
	// Names lists all flag strings (e.g., "--spec", "-s").
	Names []string
	// Type is the flag type (BoolFlag or StringFlag).
	Type FlagType
}

// FlagSet defines a collection of named flags for a command.
type FlagSet map[string]FlagDef

// ParseResult holds the parsed flag values.
type ParseResult struct {
	Bools   map[string]bool
	Strings map[string]string
	Counts  map[string]int
	Err     error // first unknown argument or missing string value
}

// Bool returns the boolean value for a flag, defaulting to false.
func (r *ParseResult) Bool(name string) bool {
	return r.Bools[name]
}

// String returns the string value for a flag, defaulting to empty string.
func (r *ParseResult) String(name string) string {
	return r.Strings[name]
}

// Count returns the number of occurrences across all aliases of a flag.
func (r *ParseResult) Count(name string) int {
	return r.Counts[name]
}

func (r *ParseResult) recordError(err error) {
	if r.Err == nil {
		r.Err = err
	}
}

// ParseFlags parses CLI arguments against a FlagSet.
// String values must be nonempty, space-separated tokens.
// Err reports the first invalid argument; parsing continues to collect recognized flags.
func ParseFlags(args []string, flags FlagSet) *ParseResult {
	result := &ParseResult{
		Bools:   make(map[string]bool),
		Strings: make(map[string]string),
		Counts:  make(map[string]int),
	}

	// Build lookup: flag string → (key, type)
	type entry struct {
		key  string
		kind FlagType
	}
	lookup := make(map[string]entry)
	for key, def := range flags {
		for _, name := range def.Names {
			lookup[name] = entry{key: key, kind: def.Type}
		}
	}

	for i := 0; i < len(args); i++ {
		e, ok := lookup[args[i]]
		if !ok {
			result.recordError(fmt.Errorf("unknown argument %q", args[i]))
			continue
		}
		result.Counts[e.key]++
		switch e.kind {
		case BoolFlag:
			result.Bools[e.key] = true
		case StringFlag:
			name := args[i]
			// Only consume the next token as a value when it does not look like
			// a flag. Tokens that start with "--" or with "-" followed by a
			// letter (e.g. "-l") are treated as flags; tokens like "-5" (dash
			// then digit) are valid negative-number values and are consumed.
			if i+1 < len(args) && !looksLikeFlag(args[i+1]) {
				i++
				result.Strings[e.key] = args[i]
				if args[i] != "" {
					continue
				}
			}
			result.recordError(fmt.Errorf("%s requires a value", name))
		}
	}

	return result
}

// looksLikeFlag reports whether s is a flag token rather than a plain value.
// "--anything" and "-<letter>" patterns are flags; "-5" (dash+digit) is a value.
func looksLikeFlag(s string) bool {
	if strings.HasPrefix(s, "--") {
		return true
	}
	if len(s) >= 2 && s[0] == '-' {
		c := s[1]
		return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	}
	return false
}
