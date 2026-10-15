package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
)

// ScheduleFields are the schedule fields a user may write. Each is also the
// name of its flag, so a flag reads the same as the control host API field.
var ScheduleFields = []string{"slug", "cron_expression", "timezone", "task_text", "custom_context", "role", "enabled"}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// RunSchedule runs `spekk schedule <subcommand> <agent> [<schedule>] [--<field> <value>...]`.
// Results go to stdout.
func RunSchedule(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("missing subcommand: use list, create, update, delete, or run")
	}
	sub := args[0]
	positional, fields, err := parseScheduleArgs(args[1:])
	if err != nil {
		return err
	}

	want := map[string]int{"list": 1, "create": 1, "update": 2, "delete": 2, "run": 2}
	n, ok := want[sub]
	if !ok {
		return fmt.Errorf("unknown schedule command %q: use list, create, update, delete, or run", sub)
	}
	if len(positional) != n {
		usage := "<agent>"
		if n == 2 {
			usage += " <schedule>"
		}
		return fmt.Errorf("usage: spekk schedule %s %s", sub, usage)
	}
	if len(fields) > 0 && sub != "create" && sub != "update" {
		return fmt.Errorf("spekk schedule %s takes no field flags", sub)
	}
	if sub == "update" && len(fields) == 0 {
		return fmt.Errorf("nothing to update: give at least one of --%s", strings.Join(ScheduleFields, ", --"))
	}
	body, err := scheduleBody(fields)
	if err != nil {
		return err
	}

	c, err := newControlHost()
	if err != nil {
		return err
	}
	agentID, err := resolveAgent(positional[0])
	if err != nil {
		return err
	}
	base := "/api/agents/" + url.PathEscape(agentID) + "/schedules/"

	if sub == "list" {
		return listSchedules(c, base, stdout)
	}
	if sub == "create" {
		var created map[string]any
		if err := c.do(http.MethodPost, base, body, &created); err != nil {
			return err
		}
		return printJSON(stdout, created)
	}

	scheduleID, err := resolveSchedule(c, base, positional[1])
	if err != nil {
		return err
	}
	path := base + url.PathEscape(scheduleID) + "/"
	switch sub {
	case "update":
		var updated map[string]any
		if err := c.do(http.MethodPatch, path, body, &updated); err != nil {
			return err
		}
		return printJSON(stdout, updated)
	case "delete":
		if err := c.do(http.MethodDelete, path, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Deleted schedule %s.\n", positional[1])
		return nil
	default: // run
		var run struct {
			MessageID string `json:"message_id"`
		}
		if err := c.do(http.MethodPost, path+"trigger/", nil, &run); err != nil {
			var apiErr *apiError
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict {
				return fmt.Errorf("schedule %s did not run: %s", positional[1], apiErr.Detail)
			}
			return err
		}
		fmt.Fprintf(stdout, "Triggered schedule %s: message %s\n", positional[1], run.MessageID)
		return nil
	}
}

// parseScheduleArgs splits positional arguments from --<field> <value>
// pairs. A value may be empty, so --custom_context "" clears the field.
func parseScheduleArgs(args []string) (positional []string, fields map[string]string, err error) {
	fields = map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			positional = append(positional, a)
			continue
		}
		name := strings.TrimPrefix(a, "--")
		if !slices.Contains(ScheduleFields, name) {
			return nil, nil, fmt.Errorf("unknown flag %s: the fields are --%s", a, strings.Join(ScheduleFields, ", --"))
		}
		if _, dup := fields[name]; dup {
			return nil, nil, fmt.Errorf("%s must be supplied only once", a)
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("%s requires a value", a)
		}
		i++
		fields[name] = args[i]
	}
	return positional, fields, nil
}

// scheduleBody turns the flags into a request body. enabled is the one field
// that is not text.
func scheduleBody(fields map[string]string) (map[string]any, error) {
	body := map[string]any{}
	for k, v := range fields {
		body[k] = v
	}
	if v, ok := fields["enabled"]; ok {
		switch v {
		case "true":
			body["enabled"] = true
		case "false":
			body["enabled"] = false
		default:
			return nil, fmt.Errorf("invalid --enabled %q: use true or false", v)
		}
	}
	return body, nil
}

// resolveAgent finds the agent id for a recorded sandbox name, or takes an
// agent id as it is, for an agent that this machine did not create.
func resolveAgent(ref string) (string, error) {
	meta, err := GetSandbox(ref)
	if err != nil {
		return "", err
	}
	if meta != nil {
		if meta.AgentID == "" {
			return "", fmt.Errorf("sandbox %q is not registered on the control host. Register it with: spekk sandbox register %s", ref, ref)
		}
		return meta.AgentID, nil
	}
	if uuidRe.MatchString(ref) {
		return ref, nil
	}
	return "", fmt.Errorf("%q is not a recorded sandbox or an agent id: give a sandbox name from \"spekk sandbox list\", or an agent id", ref)
}

// resolveSchedule takes a schedule id as it is, or finds the schedule with
// that slug. A slug is unique for each agent.
func resolveSchedule(c *controlHost, base, ref string) (string, error) {
	if uuidRe.MatchString(ref) {
		return ref, nil
	}
	all, err := fetchSchedules(c, base)
	if err != nil {
		return "", err
	}
	for _, s := range all {
		if s.Slug == ref {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("the agent has no schedule with the id or slug %q", ref)
}

// scheduleRow is what list shows of a schedule.
type scheduleRow struct {
	ID             string  `json:"id"`
	Slug           string  `json:"slug"`
	CronExpression string  `json:"cron_expression"`
	Timezone       string  `json:"timezone"`
	Role           string  `json:"role"`
	Enabled        bool    `json:"enabled"`
	NextFireAt     *string `json:"next_fire_at"`
}

// fetchSchedules reads every page. It asks for the next page by number, and
// does not follow the absolute "next" URL, so the API token goes only to
// SPEKK_HOST.
func fetchSchedules(c *controlHost, base string) ([]scheduleRow, error) {
	var all []scheduleRow
	for page := 1; ; page++ {
		var resp struct {
			Next    *string       `json:"next"`
			Results []scheduleRow `json:"results"`
		}
		if err := c.do(http.MethodGet, fmt.Sprintf("%s?page=%d", base, page), nil, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if resp.Next == nil || len(resp.Results) == 0 {
			return all, nil
		}
	}
}

func listSchedules(c *controlHost, base string, stdout io.Writer) error {
	all, err := fetchSchedules(c, base)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		fmt.Fprintln(stdout, "No schedules found.")
		return nil
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSLUG\tCRON\tTIMEZONE\tROLE\tENABLED\tNEXT FIRE")
	for _, s := range all {
		next := "-"
		if s.NextFireAt != nil {
			next = *s.NextFireAt
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%t\t%s\n", s.ID, s.Slug, s.CronExpression, s.Timezone, s.Role, s.Enabled, next)
	}
	return tw.Flush()
}

func printJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
