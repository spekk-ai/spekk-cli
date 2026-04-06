// Package status implements the `spekk status` command display.
package status

import (
	"fmt"
	"sort"

	"github.com/spekk-dev/spekk-cli/internal/parser"
)

var statusIcons = map[string]string{
	"done":        "✅",
	"in_progress": "🚧",
	"not_started": "📋",
	"blocked":     "⏸️",
	"draft":       "📝",
}

func icon(status string) string {
	if s, ok := statusIcons[status]; ok {
		return s
	}
	return statusIcons["not_started"]
}

// Show displays the status overview for all specs and assertions.
func Show(specsDir string) error {
	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return err
	}

	if len(result.Specs) == 0 && len(result.Assertions) == 0 {
		fmt.Println("📋 No specifications found in specs/ directory")
		fmt.Println()
		fmt.Println("To get started, create a spec file following the pattern:")
		fmt.Println("  specs/{spec-name}/{spec-name}.md")
		fmt.Println("  specs/{spec-name}/assertions/")
		return nil
	}

	fmt.Println("📊 Spekk Status Overview")
	fmt.Println("========================")
	fmt.Println()

	// Sort specs by priority then id.
	sortedSpecs := make([]parser.Spec, len(result.Specs))
	copy(sortedSpecs, result.Specs)
	sort.Slice(sortedSpecs, func(i, j int) bool {
		if sortedSpecs[i].Priority != sortedSpecs[j].Priority {
			return sortedSpecs[i].Priority < sortedSpecs[j].Priority
		}
		return sortedSpecs[i].ID < sortedSpecs[j].ID
	})

	for _, spec := range sortedSpecs {
		// Gather and sort assertions for this spec.
		var specAssertions []parser.Assertion
		for _, a := range result.Assertions {
			if a.Parent == spec.ID {
				specAssertions = append(specAssertions, a)
			}
		}

		sort.Slice(specAssertions, func(i, j int) bool {
			if specAssertions[i].Priority != specAssertions[j].Priority {
				return specAssertions[i].Priority < specAssertions[j].Priority
			}
			if specAssertions[i].Created != specAssertions[j].Created {
				return specAssertions[i].Created < specAssertions[j].Created
			}
			return specAssertions[i].ID < specAssertions[j].ID
		})

		done := 0
		for _, a := range specAssertions {
			if a.Status == "done" {
				done++
			}
		}

		fmt.Printf("%d %s %s (%d/%d assertions complete)\n", spec.Priority, icon(spec.Status), spec.Title, done, len(specAssertions))

		for _, a := range specAssertions {
			fmt.Printf("  %d %s %s\n", a.Priority, icon(a.Status), a.Title)
		}
		fmt.Println()
	}

	// Overall statistics.
	total := len(result.Assertions)
	var doneCount, inProgressCount, notStartedCount, blockedCount int
	for _, a := range result.Assertions {
		switch a.Status {
		case "done":
			doneCount++
		case "in_progress":
			inProgressCount++
		case "not_started":
			notStartedCount++
		case "blocked":
			blockedCount++
		}
	}

	fmt.Println("📈 Overall Progress")
	fmt.Println("-------------------")
	fmt.Printf("Total: %d assertions\n", total)
	fmt.Printf("✅ Done: %d\n", doneCount)
	fmt.Printf("🚧 In Progress: %d\n", inProgressCount)
	fmt.Printf("📋 Not Started: %d\n", notStartedCount)
	if blockedCount > 0 {
		fmt.Printf("⏸️ Blocked: %d\n", blockedCount)
	}

	pct := 0
	if total > 0 {
		pct = doneCount * 100 / total
	}
	fmt.Printf("📊 Completion: %d%%\n", pct)
	fmt.Println()

	// Next priority item.
	next := parser.FindNextAssertion(result.Assertions, result.Specs, parser.FindOptions{AllBranches: true})
	if next != nil {
		var parentTitle string
		for _, s := range result.Specs {
			if s.ID == next.Parent {
				parentTitle = s.Title
				break
			}
		}
		if parentTitle == "" {
			parentTitle = next.Parent
		}

		fmt.Println("🎯 Next Priority Item")
		fmt.Println("---------------------")
		fmt.Printf("→ %s\n", next.Title)
		fmt.Printf("  Spec: %s\n", parentTitle)
		fmt.Printf("  Priority: %d\n", next.Priority)
		fmt.Printf("  Status: %s\n", icon(next.Status))
		fmt.Printf("  File: %s\n", next.File)
	} else {
		fmt.Println("🎉 All specifications complete!")
		fmt.Println("No remaining assertions to work on.")
	}

	return nil
}
