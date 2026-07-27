package observer

import (
	"fmt"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
	"github.com/spekk-ai/spekk-cli/internal/observation"
)

// The announcement message shape lives here, in code — the observer prompt
// contains no instructions for composing announcement text. The body is, in
// order: a short evidence summary, the fixed pointer line, and a severity
// warning.

// pointerLineFormat is the fixed pointer sentence. The reference slot takes
// the observation's PR URL when the frontmatter carries one; when pr: is
// absent, the branch reference (observer/<slug>) substitutes for it — the
// branch, not the PR, is the state carrier, so a missing PR never blocks an
// announcement. "Close to dismiss" follows the lifecycle convention: closing
// parks the finding; only branch deletion forgets it.
const pointerLineFormat = "Proposed fix in PR: %s — merge to accept, close to dismiss. Reply here to discuss."

// severityWarnings maps observation severity to the warning line appended to
// every announcement body. Only high and medium exist here because low never
// announces.
var severityWarnings = map[string]string{
	observation.SeverityHigh:   "⚠️ Severity: high — specs and code disagree on load-bearing behavior; review soon.",
	observation.SeverityMedium: "⚠️ Severity: medium — meaningful drift; review when convenient.",
}

// conversationSeverities maps observation severity to the conversation
// contract's severity levels.
var conversationSeverities = map[string]conversation.Severity{
	observation.SeverityHigh:   conversation.SeverityCritical,
	observation.SeverityMedium: conversation.SeverityWarning,
}

// composeRequest builds the conversation-open request for a candidate.
func composeRequest(c Candidate) conversation.Request {
	reference := c.PR
	if reference == "" {
		reference = observation.BranchName(c.Slug)
	}

	var b strings.Builder
	b.WriteString(evidenceSummary(c.Body))
	b.WriteString("\n\nEvidence: ")
	b.WriteString(strings.Join(c.Affected, ", "))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(pointerLineFormat, reference))
	b.WriteString("\n\n")
	b.WriteString(severityWarnings[c.Severity])

	return conversation.Request{
		Title:    c.Title,
		Body:     b.String(),
		Severity: conversationSeverities[c.Severity],
	}
}

// maxSummarySentences caps the evidence summary at the spec's 2–3 sentences.
const maxSummarySentences = 3

// evidenceSummary extracts a short summary from the observation's markdown
// body: the first paragraph of the "## Issue Description" section when
// present, otherwise the first non-heading paragraph, capped at
// maxSummarySentences sentences.
func evidenceSummary(body string) string {
	paragraph := firstParagraph(sectionAfter(body, "## Issue Description"))
	if paragraph == "" {
		paragraph = firstParagraph(body)
	}
	if paragraph == "" {
		return "(no description in observation body)"
	}
	return capSentences(paragraph, maxSummarySentences)
}

// sectionAfter returns the body content following the given heading line, or
// "" when the heading is absent.
func sectionAfter(body, heading string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return ""
}

// firstParagraph returns the first run of non-empty, non-heading lines as a
// single space-joined string.
func firstParagraph(s string) string {
	var got []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(got) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if len(got) > 0 {
				break
			}
			continue
		}
		got = append(got, trimmed)
	}
	return strings.Join(got, " ")
}

// capSentences truncates s after at most n sentence terminators.
func capSentences(s string, n int) string {
	count := 0
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' {
			// Treat a terminator followed by end-of-string or a space as a
			// sentence boundary.
			if i+1 >= len(s) || s[i+1] == ' ' {
				count++
				if count >= n {
					return s[:i+1]
				}
			}
		}
	}
	return s
}
