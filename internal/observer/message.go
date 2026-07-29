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
//
// The message does not show the observation's `affected` paths. Evidence
// stays a validity condition. Selection removes a candidate that has no
// affected path. The observation file and the PR body keep the paths. A
// reader who wants the paths finds them there, with their context. A path
// list in a chat message is too long. It repeats the PR content. It also
// moves the pointer line out of view.

// pointerLineFormat is the fixed pointer sentence. The reference slot takes
// the observation's PR URL when the frontmatter carries one; when pr: is
// absent, the branch reference (observer/<slug>) substitutes for it — the
// branch, not the PR, is the state carrier, so a missing PR never blocks an
// announcement. "Close to dismiss" follows the lifecycle convention: closing
// parks the finding; only branch deletion forgets it.
const pointerLineFormat = "Proposed fix in PR: %s — merge to accept, close to dismiss. Reply here to discuss."

// batchPointerLineFormat is the per-finding pointer inside a multi-finding
// message. The "Reply here to discuss." sentence moves to the shared footer,
// so it appears once per message.
const batchPointerLineFormat = "Proposed fix in PR: %s — merge to accept, close to dismiss."

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

// reference returns the pointer target for a candidate: its PR URL when the
// frontmatter carries one, else the branch name.
func reference(c Candidate) string {
	if c.PR != "" {
		return c.PR
	}
	return observation.BranchName(c.Slug)
}

// composeBatch builds the ONE conversation-open request of an announce run.
// A single finding keeps the original message shape. Several findings share
// one message: a numbered section per finding (summary, evidence, pointer),
// then one footer with the reply invitation and the warning line of the
// highest severity present. The candidates arrive already ordered (high
// before medium, oldest first), and the sections keep that order.
func composeBatch(cands []Candidate) conversation.Request {
	if len(cands) == 1 {
		return composeSingle(cands[0])
	}

	high, medium := 0, 0
	for _, c := range cands {
		if c.Severity == observation.SeverityHigh {
			high++
		} else {
			medium++
		}
	}
	var counts []string
	if high > 0 {
		counts = append(counts, fmt.Sprintf("%d high", high))
	}
	if medium > 0 {
		counts = append(counts, fmt.Sprintf("%d medium", medium))
	}
	title := fmt.Sprintf("Observer: %d findings (%s)", len(cands), strings.Join(counts, ", "))

	var b strings.Builder
	for i, c := range cands {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("%d. *%s* (%s)\n", i+1, c.Title, c.Severity))
		b.WriteString(evidenceSummary(c.Body))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(batchPointerLineFormat, reference(c)))
	}
	b.WriteString("\n\nReply here to discuss.\n")
	top := cands[0].Severity // ordered input: the first carries the highest
	b.WriteString(severityWarnings[top])

	return conversation.Request{
		Title:    title,
		Body:     b.String(),
		Severity: conversationSeverities[top],
	}
}

// composeSingle builds the one-finding message shape.
func composeSingle(c Candidate) conversation.Request {
	var b strings.Builder
	b.WriteString(evidenceSummary(c.Body))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(pointerLineFormat, reference(c)))
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
