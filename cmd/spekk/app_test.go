package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/appclient"
)

func dropResponse(w http.ResponseWriter) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}
	_ = conn.Close()
}

func TestLaunchRecoversLostResponseWithoutSecondPost(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	posts := 0
	var requestID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			requestID, _ = body["request_id"].(string)
			dropResponse(w)
			return
		}
		_, _ = fmt.Fprintf(w, `{"id":"launch-1","request_id":%q,"stage":"queued","finished":null}`, requestID)
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := appLaunch(context.Background(), client, server.URL, map[string]any{"organization": "org-1", "audience": "personal", "size": "small", "region": "east", "kind": "one-off"}, "", "", &output, true); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || requestID == "" || !strings.Contains(output.String(), `"launch_id":"launch-1"`) || !strings.Contains(output.String(), "Sign in to Claude") {
		t.Fatalf("launch recovery failed: posts=%d, output=%s", posts, output.String())
	}
	ops, err := appclient.ListOperations()
	if err != nil || len(ops) != 1 || ops[0].RequestID != requestID {
		t.Fatalf("request ID was not saved before network I/O: %+v, %v", ops, err)
	}
}

func TestDestroyUsesStatusAfterLostResponseAndKeepsCostWarning(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts++
			dropResponse(w)
			return
		}
		_, _ = w.Write([]byte(`{"id":"launch-1","stage":"destroying","finished":null}`))
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := appCleanup(context.Background(), client, "launch-1", "destroy", &output, true); err == nil {
		t.Fatal("lost destroy response was treated as confirmed")
	}
	if posts != 1 || !strings.Contains(output.String(), "can still cost money") {
		t.Fatalf("cleanup recovery failed: posts=%d, output=%s", posts, output.String())
	}
	failed := launchResult(map[string]any{"stage": "failed", "finished": nil})
	if failed["warning"] == nil {
		t.Fatal("failed launch with unfinished cleanup lost its warning")
	}
	complete := launchResult(map[string]any{"stage": "destroyed", "finished": "2026-10-23T00:00:00Z"})
	if complete["warning"] != nil {
		t.Fatal("finished destroyed launch still warns")
	}
}

func TestChangedRequestBodyKeepsConflict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lookups := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			lookups++
			_, _ = w.Write([]byte(`{"id":"old-launch","stage":"online"}`))
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"detail":"This request ID has different launch fields."}`))
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = appLaunch(context.Background(), client, server.URL, map[string]any{"organization": "org-1", "audience": "team", "size": "small", "region": "east", "kind": "one-off"}, "", "manual-1", &output, true)
	if err == nil || !strings.Contains(err.Error(), "HTTP 409") || lookups != 0 || output.Len() != 0 {
		t.Fatalf("conflict was hidden: error=%v, lookups=%d, output=%q", err, lookups, output.String())
	}
}

func TestSendRecoversLostResponseWithoutSecondPost(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	posts := 0
	var requestID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			requestID, _ = body["request_id"].(string)
			dropResponse(w)
			return
		}
		if strings.Contains(r.URL.Path, "/chat/requests/") {
			_, _ = fmt.Fprintf(w, `{"message_id":"message-1","request_id":%q,"state":"pending"}`, requestID)
			return
		}
		_, _ = w.Write([]byte(`{"id":"launch-1","stage":"online","agent":"agent-1"}`))
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := appSend(context.Background(), client, server.URL, "launch-1", "", "run", "", "", &output, true); err != nil {
		t.Fatal(err)
	}
	if posts != 1 || !strings.Contains(output.String(), `"message_id":"message-1"`) {
		t.Fatalf("send recovery failed: posts=%d, output=%s", posts, output.String())
	}
	if id, err := appclient.TaskRequestID(requestID); err != nil || id != requestID {
		t.Fatalf("the app chat routes need a lowercase UUID request ID, got %q", requestID)
	}
	ops, err := appclient.ListOperations()
	if err != nil || len(ops) != 1 || ops[0].RequestID != requestID {
		t.Fatalf("send request was not saved: %+v, %v", ops, err)
	}
}

func TestUnconfirmedSendShowsSavedStatusWithoutSecondPost(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"detail":"This send is saved, but dispatch is not confirmed. Check saved messages before you start a new send."}`))
		case strings.Contains(r.URL.Path, "/chat/requests/"):
			_, _ = w.Write([]byte(`{"message_id":"message-1","state":"pending"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"launch-1","stage":"online","agent":"agent-1"}`))
		}
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = appSend(context.Background(), client, server.URL, "launch-1", "", "run", "", "9F2C1E6A-3B4D-4E5F-8A7B-0C1D2E3F4A5B", &output, true)
	if err == nil || !strings.Contains(err.Error(), "HTTP 409") || posts != 1 || !strings.Contains(output.String(), `"message_id":"message-1"`) {
		t.Fatalf("409 did not show the saved status once: error=%v, posts=%d, output=%s", err, posts, output.String())
	}
}

func TestGrantedMemberCanSendByManagedAgent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	launchReads := 0
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/agents/agent-1/" {
			_, _ = w.Write([]byte(`{"id":"agent-1","launch":{"id":"launch-1","stage":"online"}}`))
			return
		}
		if r.Method == "POST" && r.URL.Path == "/api/agents/agent-1/chat/" {
			posts++
			_, _ = w.Write([]byte(`{"message_id":"message-1"}`))
			return
		}
		launchReads++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	client, err := appclient.New(appclient.Config{Host: server.URL, Key: "key"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := appSend(context.Background(), client, server.URL, "", "agent-1", "run", "", "", &output, true); err != nil {
		t.Fatal(err)
	}
	if launchReads != 0 || posts != 1 || !strings.Contains(output.String(), `"launch_id":"launch-1"`) || strings.Contains(output.String(), `"request_id":null`) {
		t.Fatalf("granted member path failed: launch reads=%d, posts=%d, output=%s", launchReads, posts, output.String())
	}
}
