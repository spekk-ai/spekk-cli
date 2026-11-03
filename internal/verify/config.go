package verify

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// ErrInvalidConfig means that Run already wrote all config problems to stderr.
var ErrInvalidConfig = errors.New("invalid spekk.toml")

// ErrStrictFailed means that Run wrote the strictness errors to stderr.
var ErrStrictFailed = errors.New("strictness failed")

type runConfig struct{ command, junit string }

// Config holds the run commands and the CI rules from spekk.toml.
type Config struct {
	Runs      []runConfig
	Unchecked string
	MinLevel  map[string]string
}

// ReadConfig parses and checks the project config. Each problem is one line.
func ReadConfig(path string) (Config, []string) {
	cfg := Config{Unchecked: "warn", MinLevel: map[string]string{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, []string{err.Error()}
	}
	var raw map[string]any
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return cfg, []string{strings.ReplaceAll(err.Error(), "\n", " ")}
	}
	var problems []string
	for _, key := range keys(raw) {
		switch key {
		case "verify":
			v, ok := table(raw[key], "verify", &problems)
			if !ok {
				continue
			}
			for _, sub := range keys(v) {
				if sub != "run" {
					problems = append(problems, "unknown key verify."+sub)
					continue
				}
				runs, ok := v[sub].([]map[string]any)
				if !ok {
					problems = append(problems, "verify.run must be an array of tables")
					continue
				}
				for i, run := range runs {
					entry := runConfig{}
					for _, field := range keys(run) {
						value, valid := run[field].(string)
						if field != "command" && field != "junit" {
							problems = append(problems, fmt.Sprintf("unknown key verify.run[%d].%s", i+1, field))
						} else if !valid {
							problems = append(problems, fmt.Sprintf("verify.run[%d].%s must be a string", i+1, field))
						} else if field == "command" {
							entry.command = value
						} else {
							entry.junit = value
						}
					}
					if entry.command == "" || entry.junit == "" {
						problems = append(problems, fmt.Sprintf("verify.run[%d] needs command and junit", i+1))
					}
					cfg.Runs = append(cfg.Runs, entry)
				}
			}
		case "strict":
			v, ok := table(raw[key], "strict", &problems)
			if !ok {
				continue
			}
			for _, sub := range keys(v) {
				switch sub {
				case "unchecked":
					value, ok := v[sub].(string)
					if !ok || (value != "warn" && value != "error") {
						problems = append(problems, "strict.unchecked must be warn or error")
					} else {
						cfg.Unchecked = value
					}
				case "min-level":
					levels, ok := table(v[sub], "strict.min-level", &problems)
					if !ok {
						continue
					}
					for _, kind := range keys(levels) {
						if !parser.IsPropertyKind(kind) {
							problems = append(problems, "unknown property kind strict.min-level."+kind)
							continue
						}
						level, ok := levels[kind].(string)
						if !ok || strength(level) == 0 {
							problems = append(problems, "invalid evidence level strict.min-level."+kind)
						} else {
							cfg.MinLevel[kind] = level
						}
					}
				default:
					problems = append(problems, "unknown key strict."+sub)
				}
			}
		default:
			problems = append(problems, "unknown table or key "+key)
		}
	}
	return cfg, problems
}

func table(value any, name string, problems *[]string) (map[string]any, bool) {
	v, ok := value.(map[string]any)
	if !ok {
		*problems = append(*problems, name+" must be a table")
	}
	return v, ok
}

func keys(v map[string]any) []string {
	out := make([]string, 0, len(v))
	for key := range v {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
