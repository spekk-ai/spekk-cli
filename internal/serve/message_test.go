package serve

import (
	"strings"
	"testing"
)

func TestFormatChatMessageTextOnly(t *testing.T) {
	result := formatChatMessage(chatData{Content: "Hello world"})
	if result != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", result)
	}
}

func TestFormatChatMessageWithAttachments(t *testing.T) {
	result := formatChatMessage(chatData{
		Content: "Check this element",
		Attachments: []attachment{
			{
				Type:     "element_selection",
				Selector: "button.submit",
				Tag:      "button",
				Classes:  []string{"submit"},
			},
		},
	})
	if !strings.Contains(result, "Check this element") {
		t.Error("should contain user message")
	}
	if !strings.Contains(result, "Visual context:") {
		t.Error("should contain visual context header")
	}
	if !strings.Contains(result, "button.submit") {
		t.Error("should contain selector")
	}
}

func TestFormatElementSelection(t *testing.T) {
	result := formatElementSelection(elementSelectionData{
		Selector:    "div.main",
		Tag:         "div",
		Classes:     []string{"main", "content"},
		ID:          "app",
		InnerText:   "Hello",
		BoundingBox: &boundingBox{Width: 100, Height: 50},
	})
	if !strings.Contains(result, "`div.main`") {
		t.Error("should contain selector")
	}
	if !strings.Contains(result, "(div.main.content#app)") {
		t.Error("should contain tag description")
	}
	if !strings.Contains(result, `text: "Hello"`) {
		t.Error("should contain inner text")
	}
	if !strings.Contains(result, "dimensions: 100x50") {
		t.Error("should contain dimensions")
	}
}

func TestFormatElementSelectionTruncatesLongText(t *testing.T) {
	longText := strings.Repeat("x", 150)
	result := formatElementSelection(elementSelectionData{
		Selector:  "p",
		Tag:       "p",
		InnerText: longText,
	})
	if !strings.Contains(result, "...") {
		t.Error("should truncate long text")
	}
}

func TestFormatActionRecordingEmpty(t *testing.T) {
	result := formatActionRecording(actionRecordingData{})
	if result != "Recorded actions: (none)" {
		t.Errorf("expected 'Recorded actions: (none)', got %q", result)
	}
}

func TestFormatActionRecording(t *testing.T) {
	result := formatActionRecording(actionRecordingData{
		Actions: []action{
			{Type: "click", Selector: "#btn"},
			{Type: "type", Selector: "#input", Value: "hello"},
			{Type: "navigate", URL: "https://example.com"},
		},
	})
	if !strings.Contains(result, "Clicked on `#btn`") {
		t.Error("should format click action")
	}
	if !strings.Contains(result, `Typed "hello" in`) {
		t.Error("should format type action")
	}
	if !strings.Contains(result, "Navigated to https://example.com") {
		t.Error("should format navigate action")
	}
}

func TestFormatInitMessage(t *testing.T) {
	result := formatInitMessage(initData{
		URL:     "https://example.com",
		Title:   "Test Page",
		Version: "1.0.0",
	})
	if !strings.Contains(result, "[Session initialized]") {
		t.Error("should contain session initialized")
	}
	if !strings.Contains(result, "https://example.com") {
		t.Error("should contain URL")
	}
	if !strings.Contains(result, "Test Page") {
		t.Error("should contain title")
	}
	if !strings.Contains(result, "1.0.0") {
		t.Error("should contain version")
	}
}

func TestFormatSingleActionTypes(t *testing.T) {
	tests := []struct {
		action   action
		contains string
	}{
		{action{Type: "click", Selector: "#btn"}, "Clicked on `#btn`"},
		{action{Type: "dblclick", Selector: "#btn"}, "Double-clicked on `#btn`"},
		{action{Type: "focus", Selector: "#input"}, "Focused on `#input`"},
		{action{Type: "blur", Selector: "#input"}, "Left `#input`"},
		{action{Type: "hover", Selector: ".link"}, "Hovered over `.link`"},
		{action{Type: "submit", Selector: "#form"}, "Submitted form `#form`"},
		{action{Type: "scroll", Selector: "#div", Direction: "down"}, "Scrolled down `#div`"},
		{action{Type: "keydown", Selector: "#input", Key: "Enter"}, `Pressed "Enter"`},
		{action{Type: "select", Selector: "#dropdown", Value: "option1"}, `Selected "option1"`},
		{action{Type: "change", Selector: "#field", Value: "new"}, "Changed `#field` to \"new\""},
		{action{Type: "navigate", URL: "http://test.com"}, "Navigated to http://test.com"},
		{action{Type: "navigate"}, "Navigated to new page"},
		{action{Type: "custom", Selector: "#el"}, "custom on `#el`"},
	}

	for _, tt := range tests {
		result := formatSingleAction(tt.action)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("action type %q: expected result to contain %q, got %q", tt.action.Type, tt.contains, result)
		}
	}
}

func TestFormatAttachmentScreenshot(t *testing.T) {
	tests := []struct {
		att      attachment
		contains string
	}{
		{attachment{Type: "screenshot", ElementSelector: "#el", Description: "desc"}, "[Screenshot of `#el`: desc]"},
		{attachment{Type: "screenshot", ElementSelector: "#el"}, "[Screenshot of `#el`]"},
		{attachment{Type: "screenshot", Description: "desc"}, "[Screenshot: desc]"},
		{attachment{Type: "screenshot"}, "[Screenshot of current page]"},
		{attachment{Type: "unknown"}, "[Attachment: unknown]"},
	}

	for _, tt := range tests {
		result := formatAttachment(tt.att)
		if result != tt.contains {
			t.Errorf("expected %q, got %q", tt.contains, result)
		}
	}
}

func TestDescribeToolUse(t *testing.T) {
	tests := []struct {
		tool     string
		expected string
	}{
		{"Read", "Reading files..."},
		{"Write", "Writing code..."},
		{"Bash", "Running a command..."},
		{"WebSearch", "Searching the web..."},
		{"CustomTool", "Using CustomTool..."},
		{"", "Working..."},
	}

	for _, tt := range tests {
		result := describeToolUse(tt.tool)
		if result != tt.expected {
			t.Errorf("describeToolUse(%q) = %q, want %q", tt.tool, result, tt.expected)
		}
	}
}
