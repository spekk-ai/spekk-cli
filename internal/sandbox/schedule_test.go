package sandbox

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// scheduleSetup records a registered sandbox "box" and returns its agent id.
func scheduleSetup(t *testing.T) (*fakeControlHost, string) {
	t.Helper()
	isolateConfig(t)
	useTempStore(t)
	host := useControlHost(t)
	id := host.addAgent("box")
	if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", AgentID: id}); err != nil {
		t.Fatal(err)
	}
	return host, id
}

func runSchedule(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSchedule(args, &out)
	return out.String(), err
}

func TestScheduleLifecycle(t *testing.T) {
	host, agentID := scheduleSetup(t)

	out, err := runSchedule(t, "create", "box", "--slug", "nightly-scan", "--cron_expression", "0 9 * * *", "--task_text", "Scan the specs", "--enabled", "false")
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("create must print the schedule as JSON: %v\n%s", err, out)
	}
	if created["slug"] != "nightly-scan" || created["enabled"] != false {
		t.Errorf("created = %v", created)
	}
	// Only the given fields go to the control host.
	var sent map[string]any
	json.Unmarshal([]byte(host.requests[len(host.requests)-1].Body), &sent)
	if len(sent) != 4 {
		t.Errorf("create sent %v, want the four given fields only", sent)
	}

	// The slug finds the schedule, and an empty value clears a field.
	if _, err := runSchedule(t, "update", "box", "nightly-scan", "--custom_context", ""); err != nil {
		t.Fatal(err)
	}
	last := host.requests[len(host.requests)-1]
	if last.Method != "PATCH" || last.Body != `{"custom_context":""}` {
		t.Errorf("update sent %s %s", last.Method, last.Body)
	}

	if out, err = runSchedule(t, "run", "box", "nightly-scan"); err != nil || !strings.Contains(out, "message") {
		t.Errorf("run: %q, %v", out, err)
	}
	host.triggerStatus = 409
	if _, err = runSchedule(t, "run", agentID, "nightly-scan"); err == nil || !strings.Contains(err.Error(), "is not connected") {
		t.Errorf("a 409 must show the control host's reason, got %v", err)
	}

	if _, err := runSchedule(t, "delete", "box", "nightly-scan"); err != nil {
		t.Fatal(err)
	}
	if out, _ := runSchedule(t, "list", "box"); !strings.Contains(out, "No schedules") {
		t.Errorf("list after delete = %q", out)
	}
}

func TestScheduleListReadsEveryPage(t *testing.T) {
	host, _ := scheduleSetup(t)
	host.pageSize = 2
	for _, slug := range []string{"a-job", "b-job", "c-job"} {
		if _, err := runSchedule(t, "create", "box", "--slug", slug, "--cron_expression", "0 9 * * *", "--task_text", "x"); err != nil {
			t.Fatal(err)
		}
	}
	out, err := runSchedule(t, "list", "box")
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"a-job", "b-job", "c-job", "0 9 * * *", "NEXT FIRE"} {
		if !strings.Contains(out, slug) {
			t.Errorf("list lacks %q:\n%s", slug, out)
		}
	}
}

func TestScheduleRefusesBadInput(t *testing.T) {
	scheduleSetup(t)
	if err := SaveSandbox("plain", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4"}); err != nil {
		t.Fatal(err)
	}
	for want, args := range map[string][]string{
		"spekk sandbox register plain": {"list", "plain"},
		"not a recorded sandbox":       {"list", "nobody"},
		"unknown flag --cron":          {"create", "box", "--cron", "x"},
		"true or false":                {"update", "box", "job", "--enabled", "yes"},
		"nothing to update":            {"update", "box", "job"},
		"no schedule with":             {"run", "box", "no-such-job"},
	} {
		if _, err := runSchedule(t, args...); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: want an error with %q, got %v", args, want, err)
		}
	}
}
