// Package serve implements the WebSocket server that bridges the browser
// extension to Claude Code for interactive spec coaching.
package serve

import (
	"encoding/json"
	"fmt"
	"strings"
)

// --- Incoming message types (browser → server) ---

// incomingMessage is the raw envelope from the browser extension.
// Wire protocol: { "event": "coach:<operation>", "data": { ... } }
type incomingMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type chatData struct {
	Content     string       `json:"content"`
	Attachments []attachment `json:"attachments"`
}

type attachment struct {
	Type            string      `json:"type"`
	Selector        string      `json:"selector,omitempty"`
	Tag             string      `json:"tag,omitempty"`
	Classes         []string    `json:"classes,omitempty"`
	ID              string      `json:"id,omitempty"`
	InnerText       string      `json:"inner_text,omitempty"`
	BoundingBox     *boundingBox `json:"bounding_box,omitempty"`
	ElementSelector string      `json:"element_selector,omitempty"`
	Description     string      `json:"description,omitempty"`
	Actions         []action    `json:"actions,omitempty"`
}

type boundingBox struct {
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

type action struct {
	Type      string `json:"type"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	Key       string `json:"key,omitempty"`
	URL       string `json:"url,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type elementSelectionData struct {
	Selector    string      `json:"selector"`
	Tag         string      `json:"tag"`
	Classes     []string    `json:"classes,omitempty"`
	ID          string      `json:"id,omitempty"`
	InnerText   string      `json:"inner_text,omitempty"`
	BoundingBox *boundingBox `json:"bounding_box,omitempty"`
}

type actionRecordingData struct {
	Actions []action `json:"actions"`
}

type initData struct {
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// --- Outgoing message types (server → browser) ---

type outgoingMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type statusData struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type assistantData struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"`
}

type resultData struct {
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
	SessionID string `json:"session_id"`
}

type errorData struct {
	Message string `json:"message"`
}

type agentExitedData struct {
	Code int `json:"code"`
}

// --- Formatters ---

// formatChatMessage formats a chat message with optional attachments.
func formatChatMessage(data chatData) string {
	var parts []string

	if data.Content != "" {
		parts = append(parts, data.Content)
	}

	if len(data.Attachments) > 0 {
		parts = append(parts, "") // blank line
		parts = append(parts, "Visual context:")
		for _, att := range data.Attachments {
			parts = append(parts, formatAttachment(att))
		}
	}

	return strings.Join(parts, "\n")
}

// formatElementSelection formats an element selection into readable text.
func formatElementSelection(data elementSelectionData) string {
	parts := []string{fmt.Sprintf("Selected element: `%s`", data.Selector)}

	tagParts := []string{data.Tag}
	if len(data.Classes) > 0 {
		tagParts = append(tagParts, "."+strings.Join(data.Classes, "."))
	}
	if data.ID != "" {
		tagParts = append(tagParts, "#"+data.ID)
	}
	parts = append(parts, fmt.Sprintf("(%s)", strings.Join(tagParts, "")))

	if data.InnerText != "" {
		text := data.InnerText
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		parts = append(parts, fmt.Sprintf(`text: "%s"`, text))
	}

	if data.BoundingBox != nil && data.BoundingBox.Width > 0 && data.BoundingBox.Height > 0 {
		parts = append(parts, fmt.Sprintf("dimensions: %.0fx%.0f", data.BoundingBox.Width, data.BoundingBox.Height))
	}

	return strings.Join(parts, ", ")
}

// formatActionRecording formats recorded browser actions.
func formatActionRecording(data actionRecordingData) string {
	if len(data.Actions) == 0 {
		return "Recorded actions: (none)"
	}

	lines := []string{"Recorded actions:"}
	for i, a := range data.Actions {
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, formatSingleAction(a)))
	}
	return strings.Join(lines, "\n")
}

// formatInitMessage formats a session init message.
func formatInitMessage(data initData) string {
	parts := []string{"[Session initialized]"}
	if data.URL != "" {
		parts = append(parts, "Current page: "+data.URL)
	}
	if data.Title != "" {
		parts = append(parts, "Page title: "+data.Title)
	}
	if data.Version != "" {
		parts = append(parts, "Extension version: "+data.Version)
	}
	return strings.Join(parts, "\n")
}

func formatSingleAction(a action) string {
	selector := "element"
	if a.Selector != "" {
		selector = "`" + a.Selector + "`"
	}

	switch a.Type {
	case "click":
		return "Clicked on " + selector
	case "dblclick":
		return "Double-clicked on " + selector
	case "input", "type":
		if a.Value != "" {
			return fmt.Sprintf(`Typed "%s" in %s`, a.Value, selector)
		}
		return "Typed in " + selector
	case "change":
		if a.Value != "" {
			return fmt.Sprintf(`Changed %s to "%s"`, selector, a.Value)
		}
		return "Changed " + selector
	case "select":
		if a.Value != "" {
			return fmt.Sprintf(`Selected "%s" in %s`, a.Value, selector)
		}
		return "Selected option in " + selector
	case "focus":
		return "Focused on " + selector
	case "blur":
		return "Left " + selector
	case "scroll":
		dir := a.Direction
		if dir == "" {
			dir = "on"
		}
		return fmt.Sprintf("Scrolled %s %s", dir, selector)
	case "hover":
		return "Hovered over " + selector
	case "keypress", "keydown":
		if a.Key != "" {
			return fmt.Sprintf(`Pressed "%s" in %s`, a.Key, selector)
		}
		return "Pressed key in " + selector
	case "navigate", "navigation":
		if a.URL != "" {
			return "Navigated to " + a.URL
		}
		return "Navigated to new page"
	case "submit":
		return "Submitted form " + selector
	default:
		if a.Value != "" {
			return fmt.Sprintf(`%s on %s (value: "%s")`, a.Type, selector, a.Value)
		}
		return a.Type + " on " + selector
	}
}

func formatAttachment(att attachment) string {
	switch att.Type {
	case "element_selection":
		return formatElementSelection(elementSelectionData{
			Selector:    att.Selector,
			Tag:         att.Tag,
			Classes:     att.Classes,
			ID:          att.ID,
			InnerText:   att.InnerText,
			BoundingBox: att.BoundingBox,
		})
	case "screenshot":
		if att.Description != "" && att.ElementSelector != "" {
			return fmt.Sprintf("[Screenshot of `%s`: %s]", att.ElementSelector, att.Description)
		}
		if att.ElementSelector != "" {
			return fmt.Sprintf("[Screenshot of `%s`]", att.ElementSelector)
		}
		if att.Description != "" {
			return fmt.Sprintf("[Screenshot: %s]", att.Description)
		}
		return "[Screenshot of current page]"
	case "action_recording":
		return formatActionRecording(actionRecordingData{Actions: att.Actions})
	default:
		return fmt.Sprintf("[Attachment: %s]", att.Type)
	}
}

// describeToolUse maps Claude tool names to user-friendly descriptions.
func describeToolUse(toolName string) string {
	switch toolName {
	case "Read", "Glob", "Grep":
		return "Reading files..."
	case "Write", "Edit", "NotebookEdit":
		return "Writing code..."
	case "Bash":
		return "Running a command..."
	case "WebSearch", "WebFetch":
		return "Searching the web..."
	default:
		if toolName != "" {
			return fmt.Sprintf("Using %s...", toolName)
		}
		return "Working..."
	}
}
