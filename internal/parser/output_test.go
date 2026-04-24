package parser

import (
	"encoding/json"
	"testing"
)

func TestFormatNextAssertion_MatchesNodeFormat(t *testing.T) {
	a := &Assertion{
		ID:        "test-assertion",
		Parent:    "test-spec",
		File:      "specs/test-spec/assertions/test-assertion.md",
		Priority:  1,
		Status:    "not_started",
		Branch:    "feature/test",
		Created:   "2026-01-01T00:00:00Z",
		DependsOn: "other-assertion",
		LockedBy:  "",
		Title:     "Test assertion title",
		Content:   "---\nid: test-assertion\n---\n# Test assertion title\n",
	}
	specs := []Spec{
		{ID: "test-spec", File: "specs/test-spec/test-spec.md", Title: "Test Spec"},
	}

	data, err := FormatNextAssertion(a, specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify required fields.
	expectedFields := []string{"type", "id", "parent", "file", "priority", "status", "branch", "created", "title", "content", "spec"}
	for _, f := range expectedFields {
		if _, ok := result[f]; !ok {
			t.Errorf("missing field %q", f)
		}
	}

	if result["type"] != "assertion" {
		t.Errorf("expected type=assertion, got %v", result["type"])
	}
	if result["id"] != "test-assertion" {
		t.Errorf("expected id=test-assertion, got %v", result["id"])
	}

	// Verify camelCase field names.
	if result["dependsOn"] != "other-assertion" {
		t.Errorf("expected dependsOn=other-assertion, got %v", result["dependsOn"])
	}

	// Verify spec ref.
	specRef, ok := result["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("expected spec to be an object")
	}
	if specRef["id"] != "test-spec" {
		t.Errorf("expected spec.id=test-spec, got %v", specRef["id"])
	}
	if specRef["file"] != "specs/test-spec/test-spec.md" {
		t.Errorf("expected spec.file path, got %v", specRef["file"])
	}
	if specRef["title"] != "Test Spec" {
		t.Errorf("expected spec.title=Test Spec, got %v", specRef["title"])
	}
}

func TestFormatNextAssertion_OmitsEmptyOptionalFields(t *testing.T) {
	a := &Assertion{
		ID:       "simple",
		Parent:   "spec",
		File:     "specs/spec/assertions/simple.md",
		Priority: 1,
		Status:   "not_started",
		Branch:   "main",
		Created:  "2026-01-01T00:00:00Z",
		Title:    "Simple",
		Content:  "content",
	}

	data, err := FormatNextAssertion(a, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// Empty dependsOn and lockedBy should be omitted.
	if _, ok := result["dependsOn"]; ok {
		t.Error("expected dependsOn to be omitted when empty")
	}
	if _, ok := result["lockedBy"]; ok {
		t.Error("expected lockedBy to be omitted when empty")
	}
}

func TestFormatHierarchy_Structure(t *testing.T) {
	result := &ParseResult{
		Specs: []Spec{
			{ID: "spec-b", Title: "Spec B", Status: "in_progress", Priority: 2, File: "specs/spec-b/spec-b.md"},
			{ID: "spec-a", Title: "Spec A", Status: "done", Priority: 1, File: "specs/spec-a/spec-a.md"},
		},
		Assertions: []Assertion{
			{ID: "b2", Parent: "spec-b", Title: "B2", Status: "not_started", Priority: 2, File: "specs/spec-b/assertions/b2.md"},
			{ID: "b1", Parent: "spec-b", Title: "B1", Status: "done", Priority: 1, File: "specs/spec-b/assertions/b1.md"},
			{ID: "a1", Parent: "spec-a", Title: "A1", Status: "done", Priority: 1, File: "specs/spec-a/assertions/a1.md"},
		},
	}

	data, err := FormatHierarchy(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal(data, &out)

	if out["type"] != "hierarchy" {
		t.Errorf("expected type=hierarchy, got %v", out["type"])
	}

	specs := out["specs"].([]interface{})
	// spec-a (priority 1) should come before spec-b (priority 2).
	firstSpec := specs[0].(map[string]interface{})
	if firstSpec["id"] != "spec-a" {
		t.Errorf("expected first spec to be spec-a (priority 1), got %v", firstSpec["id"])
	}

	// Check spec-b assertions are sorted by priority then id.
	secondSpec := specs[1].(map[string]interface{})
	assertions := secondSpec["assertions"].([]interface{})
	if len(assertions) != 2 {
		t.Fatalf("expected 2 assertions for spec-b, got %d", len(assertions))
	}
	firstAssertion := assertions[0].(map[string]interface{})
	if firstAssertion["id"] != "b1" {
		t.Errorf("expected first assertion b1 (priority 1), got %v", firstAssertion["id"])
	}

	// observations should be present (empty array).
	obs := out["observations"].([]interface{})
	if len(obs) != 0 {
		t.Errorf("expected empty observations array, got %d", len(obs))
	}
}

func TestFormatComplete(t *testing.T) {
	data, err := FormatComplete()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["type"] != "complete" {
		t.Errorf("expected type=complete, got %v", result["type"])
	}
	if result["status"] != "complete" {
		t.Errorf("expected status=complete, got %v", result["status"])
	}
	if result["message"] != "All specifications are complete" {
		t.Errorf("unexpected message: %v", result["message"])
	}
}

func TestFormatEmpty(t *testing.T) {
	data, err := FormatEmpty()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["status"] != "empty" {
		t.Errorf("expected status=empty, got %v", result["status"])
	}
	if result["message"] != "No specifications found in specs/ directory" {
		t.Errorf("unexpected message: %v", result["message"])
	}
}

func TestFormatError(t *testing.T) {
	data, err := FormatError("something broke")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["error"] != true {
		t.Errorf("expected error=true, got %v", result["error"])
	}
	if result["message"] != "something broke" {
		t.Errorf("unexpected message: %v", result["message"])
	}
}

func TestMarshalJSON_NoHTMLEscape(t *testing.T) {
	data := map[string]string{"title": "A & B <test>"}
	b, err := marshalJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(b)
	if s != "{\n  \"title\": \"A & B <test>\"\n}" {
		t.Errorf("expected unescaped HTML chars, got: %s", s)
	}
}
