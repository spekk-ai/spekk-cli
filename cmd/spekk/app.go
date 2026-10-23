package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/appclient"
)

const appHelp = `spekk app - Use app-managed sandboxes with a personal API key

  spekk app auth set --host https://app.example.test   (read key from stdin)
  spekk app sandbox options|repositories|agents|launch|list|status|wait|cancel|destroy|send|result|operations

Use spekk app sandbox <command> --help for its flags. These commands use the app API.
The separate spekk sandbox commands manage cloud machines directly.
`

func execApp(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		_, _ = io.WriteString(stdout, appHelp)
		return nil
	}
	if args[0] == "auth" {
		if len(args) < 2 || args[1] != "set" {
			return errors.New("use spekk app auth set --host <origin> and provide the key on stdin")
		}
		fs := appFlags("auth set", stderr)
		host := fs.String("host", "", "app origin")
		if err := checkAppArgs(args[2:]); err != nil {
			return err
		}
		if err := fs.Parse(args[2:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected auth argument %q", fs.Arg(0))
		}
		data, err := io.ReadAll(io.LimitReader(stdin, 4097))
		if err != nil {
			return err
		}
		if err := appclient.SaveConfig(*host, string(data)); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, "App host and personal key saved in a private file.")
		return nil
	}
	if args[0] != "sandbox" || len(args) < 2 {
		return errors.New("use spekk app sandbox <command>")
	}
	return execAppSandbox(args[1], args[2:], stdout, stderr)
}

func appFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func execAppSandbox(command string, args []string, stdout, stderr io.Writer) error {
	if err := checkAppArgs(args); err != nil {
		return err
	}
	fs := appFlags(command, stderr)
	jsonMode := fs.Bool("json", false, "print one JSON object")
	org := fs.String("organization", "", "organization ID")
	search := fs.String("search", "", "repository search text")
	page := fs.Int("page", 1, "repository page")
	launchID := fs.String("launch", "", "launch ID")
	requestID := fs.String("request-id", "", "saved launch or task request ID")
	agentID := fs.String("agent", "", "agent ID")
	messageID := fs.String("message", "", "message ID")
	audience := fs.String("audience", "", "team or personal")
	size := fs.String("size", "", "server size slug")
	region := fs.String("region", "", "server region")
	kind := fs.String("kind", "", "one-off or persistent")
	repository := fs.String("repository", "", "repository ID")
	text := fs.String("text", "", "task text")
	session := fs.String("session", "", "Claude session ID returned by the app")
	timeout := fs.Duration("timeout", 0, "bounded wait, such as 2m")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected app sandbox argument %q", fs.Arg(0))
	}
	if command == "operations" {
		ops, err := appclient.ListOperations()
		if err != nil {
			return err
		}
		return printApp(stdout, *jsonMode, map[string]any{"operations": ops}, "")
	}
	config, err := appclient.LoadConfig()
	if err != nil {
		return err
	}
	client, err := appclient.New(config)
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch command {
	case "options":
		return appRead(ctx, client, "/api/sandbox-launches/options/", stdout, *jsonMode)
	case "repositories":
		if *org == "" || *page < 1 || *page > 100 {
			return errors.New("repositories needs --organization and a page from 1 to 100")
		}
		query := url.Values{"organization": {*org}, "page": {fmt.Sprint(*page)}, "search": {*search}}
		return appRead(ctx, client, "/api/sandbox-launches/repositories/?"+query.Encode(), stdout, *jsonMode)
	case "list":
		return appRead(ctx, client, "/api/sandbox-launches/", stdout, *jsonMode)
	case "agents":
		return appRead(ctx, client, "/api/agents/", stdout, *jsonMode)
	case "launch":
		if *org == "" || *audience == "" || *size == "" || *region == "" || *kind == "" {
			return errors.New("launch needs --organization, --audience, --size, --region, and --kind")
		}
		return appLaunch(ctx, client, config.Host, map[string]any{"organization": *org, "audience": *audience, "size": *size, "region": *region, "kind": *kind}, *repository, *requestID, stdout, *jsonMode)
	case "status", "wait":
		if *launchID == "" && *requestID == "" {
			return errors.New("status or wait needs --launch or --request-id")
		}
		return appLaunchStatus(ctx, client, *launchID, *requestID, *timeout, command == "wait", stdout, *jsonMode)
	case "cancel", "destroy":
		if *launchID == "" {
			return errors.New("cancel or destroy needs --launch")
		}
		return appCleanup(ctx, client, *launchID, command, stdout, *jsonMode)
	case "send":
		if (*launchID == "" && *agentID == "") || strings.TrimSpace(*text) == "" {
			return errors.New("send needs --launch or --agent, and --text")
		}
		return appSend(ctx, client, config.Host, *launchID, *agentID, *text, *session, *requestID, stdout, *jsonMode)
	case "result":
		if *agentID == "" || (*messageID == "" && *requestID == "") {
			return errors.New("result needs --agent and --message or --request-id")
		}
		return appResult(ctx, client, *agentID, *messageID, *requestID, *timeout, stdout, *jsonMode)
	default:
		return fmt.Errorf("unknown app sandbox command %q", command)
	}
}

func checkAppArgs(args []string) error {
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		if strings.Contains(arg, "=") {
			return fmt.Errorf("use a space after %s to give its value", strings.SplitN(arg, "=", 2)[0])
		}
		if arg != "--json" && seen[arg] {
			return fmt.Errorf("%s was given more than once", arg)
		}
		seen[arg] = true
		if arg != "--json" && arg != "--help" {
			i++
		}
	}
	return nil
}

func appRead(ctx context.Context, client *appclient.Client, route string, stdout io.Writer, jsonMode bool) error {
	result, err := client.Do(ctx, "GET", route, nil)
	if err != nil {
		return err
	}
	return printApp(stdout, jsonMode, result, "")
}

func appLaunch(ctx context.Context, client *appclient.Client, host string, body map[string]any, repository, requestID string, stdout io.Writer, jsonMode bool) error {
	if repository != "" {
		body["repository"] = repository
	}
	hash, err := payloadHash(body)
	if err != nil {
		return err
	}
	id := requestID
	if id == "" {
		id, err = appclient.NewRequestID()
		if err != nil {
			return err
		}
	}
	body["request_id"] = id
	op := appclient.Operation{Kind: "launch", RequestID: id, Host: host, PayloadHash: hash}
	if err := appclient.SaveOperation(op); err != nil {
		return err
	}
	result, err := client.Do(ctx, "POST", "/api/sandbox-launches/", body)
	if err != nil {
		if definiteAPIError(err) {
			return err
		}
		recovered, lookupErr := client.Do(ctx, "GET", launchRoute("", id), nil)
		if lookupErr == nil {
			return printApp(stdout, jsonMode, launchResult(recovered), "spekk app sandbox status --request-id "+id)
		}
		if printErr := printApp(stdout, jsonMode, map[string]any{"request_id": id, "state": "uncertain", "error": err.Error()}, "spekk app sandbox status --request-id "+id); printErr != nil {
			return printErr
		}
		return fmt.Errorf("launch outcome is uncertain; use its saved request ID")
	}
	op.LaunchID = field(result, "id")
	if err := appclient.SaveOperation(op); err != nil {
		return err
	}
	return printApp(stdout, jsonMode, launchResult(result), "spekk app sandbox wait --launch "+op.LaunchID+" --timeout 10m")
}

func launchRoute(launchID, requestID string) string {
	if requestID != "" {
		return "/api/sandbox-launches/requests/" + url.PathEscape(requestID) + "/"
	}
	return "/api/sandbox-launches/" + url.PathEscape(launchID) + "/"
}

func appLaunchStatus(ctx context.Context, client *appclient.Client, launchID, requestID string, timeout time.Duration, wait bool, stdout io.Writer, jsonMode bool) error {
	if wait && (timeout <= 0 || timeout > 24*time.Hour) {
		return errors.New("wait needs --timeout from 1s to 24h")
	}
	if wait {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	for {
		result, err := client.Do(ctx, "GET", launchRoute(launchID, requestID), nil)
		if err != nil {
			return err
		}
		out := launchResult(result)
		stage := field(result, "stage")
		if !wait || stage == "online" || stage == "failed" || stage == "destroyed" {
			return printApp(stdout, jsonMode, out, "")
		}
		select {
		case <-ctx.Done():
			out["error"] = "wait timed out; launch continues remotely"
			if err := printApp(stdout, jsonMode, out, ""); err != nil {
				return err
			}
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func appCleanup(ctx context.Context, client *appclient.Client, launchID, command string, stdout io.Writer, jsonMode bool) error {
	route := launchRoute(launchID, "")
	result, err := client.Do(ctx, "POST", route+command+"/", map[string]any{})
	if err != nil {
		if definiteAPIError(err) {
			return err
		}
		observed, readErr := client.Do(ctx, "GET", route, nil)
		if readErr != nil {
			if printErr := printApp(stdout, jsonMode, map[string]any{"launch_id": launchID, "state": "uncertain", "error": err.Error()}, "spekk app sandbox status --launch "+launchID); printErr != nil {
				return printErr
			}
			return errors.New("cleanup outcome is uncertain; check launch status")
		}
		out := launchResult(observed)
		out["error"] = err.Error()
		if command == "cancel" && field(observed, "stage") == "failed" || command == "destroy" && field(observed, "stage") == "destroyed" {
			return printApp(stdout, jsonMode, out, "")
		}
		out["state"] = "uncertain"
		if printErr := printApp(stdout, jsonMode, out, "spekk app sandbox status --launch "+launchID); printErr != nil {
			return printErr
		}
		return errors.New("cleanup outcome is uncertain; choose any further action after checking status")
	}
	return printApp(stdout, jsonMode, launchResult(result), "spekk app sandbox status --launch "+launchID)
}

func appSend(ctx context.Context, client *appclient.Client, host, launchID, agentID, text, session, requestID string, stdout io.Writer, jsonMode bool) error {
	body := map[string]any{"text": text}
	if session != "" {
		body["session_id"] = session
	}
	hash, err := payloadHash(body)
	if err != nil {
		return err
	}
	id := requestID
	if id == "" {
		id, err = appclient.NewRequestID()
		if err != nil {
			return err
		}
	}
	op := appclient.Operation{Kind: "send", RequestID: id, Host: host, LaunchID: launchID, AgentID: agentID, PayloadHash: hash}
	if err := appclient.SaveOperation(op); err != nil {
		return err
	}
	var launch map[string]any
	if agentID == "" {
		launch, err = client.Do(ctx, "GET", launchRoute(launchID, ""), nil)
	} else {
		var agent map[string]any
		agent, err = client.Do(ctx, "GET", "/api/agents/"+url.PathEscape(agentID)+"/", nil)
		launch, _ = agent["launch"].(map[string]any)
	}
	if err != nil {
		return err
	}
	if launch == nil {
		return errors.New("agent is not an app-managed sandbox")
	}
	if agentID == "" {
		agentID = field(launch, "agent")
	} else {
		launchID = field(launch, "id")
	}
	if field(launch, "stage") != "online" || agentID == "" {
		return errors.New("launch is not online with a registered agent")
	}
	op.LaunchID = launchID
	op.AgentID = agentID
	if err := appclient.SaveOperation(op); err != nil {
		return err
	}
	body["request_id"] = id
	route := "/api/agents/" + url.PathEscape(agentID) + "/chat/"
	result, err := client.Do(ctx, "POST", route, body)
	if err != nil {
		if definiteAPIError(err) {
			return err
		}
		recovered, lookupErr := client.Do(ctx, "GET", route+"requests/"+url.PathEscape(id)+"/", nil)
		if lookupErr == nil {
			return printApp(stdout, jsonMode, chatResult(recovered, launchID, agentID), "")
		}
		if printErr := printApp(stdout, jsonMode, map[string]any{"request_id": id, "launch_id": launchID, "agent_id": agentID, "state": "uncertain", "error": err.Error()}, "spekk app sandbox result --agent "+agentID+" --request-id "+id); printErr != nil {
			return printErr
		}
		return errors.New("task outcome is uncertain; use its saved request ID")
	}
	op.MessageID = field(result, "message_id")
	if err := appclient.SaveOperation(op); err != nil {
		return err
	}
	return printApp(stdout, jsonMode, chatResult(result, launchID, agentID), "spekk app sandbox result --agent "+agentID+" --message "+op.MessageID)
}

func appResult(ctx context.Context, client *appclient.Client, agentID, messageID, requestID string, timeout time.Duration, stdout io.Writer, jsonMode bool) error {
	if timeout < 0 || timeout > 24*time.Hour {
		return errors.New("result timeout must be from 0 to 24h")
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	route := "/api/agents/" + url.PathEscape(agentID) + "/chat/"
	if requestID != "" {
		route += "requests/" + url.PathEscape(requestID) + "/"
	} else {
		route += url.PathEscape(messageID) + "/"
	}
	for {
		result, err := client.Do(ctx, "GET", route, nil)
		if err != nil {
			return err
		}
		out := chatResult(result, "", agentID)
		if out["request_id"] == nil || out["launch_id"] == "" {
			ops, err := appclient.ListOperations()
			if err != nil {
				return err
			}
			for _, op := range ops {
				if op.AgentID == agentID && (op.MessageID == field(result, "message_id") || op.RequestID == requestID && requestID != "") {
					out["request_id"] = op.RequestID
					out["launch_id"] = op.LaunchID
					break
				}
			}
		}
		state := field(result, "state")
		if timeout == 0 || state == "completed" || state == "yielded" || state == "failed" || state == "abandoned" {
			return printApp(stdout, jsonMode, out, "")
		}
		select {
		case <-ctx.Done():
			out["error"] = "result wait timed out; task continues remotely"
			if err := printApp(stdout, jsonMode, out, ""); err != nil {
				return err
			}
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func launchResult(result map[string]any) map[string]any {
	stage := field(result, "stage")
	finished := field(result, "finished")
	out := map[string]any{"request_id": result["request_id"], "launch_id": result["id"], "agent_id": result["agent"], "state": stage, "stage": stage, "finished": result["finished"], "error": result["error"], "expires_at": result["expires_at"]}
	if !((stage == "failed" || stage == "destroyed") && finished != "") {
		out["warning"] = "Cleanup is incomplete. A machine can still cost money; monitor the launch and ask an organization admin if cleanup stalls."
	}
	return out
}

func chatResult(result map[string]any, launchID, agentID string) map[string]any {
	state := field(result, "state")
	if state == "" {
		state = "pending"
	}
	return map[string]any{"request_id": result["request_id"], "launch_id": launchID, "agent_id": agentID, "message_id": result["message_id"], "agent_session_id": result["agent_session_id"], "session_id": result["session_id"], "state": state, "reply": result["reply"], "error": result["failure_reason"]}
}

func field(result map[string]any, key string) string {
	value, _ := result[key].(string)
	return value
}

func payloadHash(body map[string]any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func definiteAPIError(err error) bool {
	var apiErr *appclient.APIError
	return errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 && apiErr.Status != 408 && apiErr.Status != 429
}

func printApp(w io.Writer, jsonMode bool, result map[string]any, next string) error {
	if jsonMode {
		return json.NewEncoder(w).Encode(result)
	}
	keys := make([]string, 0, len(result))
	for key := range result {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if result[key] != nil && result[key] != "" {
			if _, err := fmt.Fprintf(w, "%s: %v\n", key, result[key]); err != nil {
				return err
			}
		}
	}
	if next != "" {
		_, _ = fmt.Fprintln(w, "Next:", next)
	}
	return nil
}
