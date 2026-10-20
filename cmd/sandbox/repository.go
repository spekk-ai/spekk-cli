package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	repositoryCredentialHelper = "/opt/spekk/agent-client git-credential"
	repositorySessionEnv       = "SPEKK_REPOSITORY_SESSION_ID"
	repositoryIDEnv            = "SPEKK_REPOSITORY_ID"
	repositoryRefEnv           = "SPEKK_REPOSITORY_REF"
	repositorySessionConfig    = "spekk.repositorySessionId"
	repositoryIDConfig         = "spekk.repositoryId"
	repositoryRefConfig        = "spekk.repositoryRef"
)

var repositoryNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
var pullRefPattern = regexp.MustCompile(`^pull/[0-9]+$`)
var repositoryLocks sync.Map

type repositoryAccess struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Token      string `json:"token"`
}

func requestRepositoryAccess(ctx context.Context, cfg Config, sessionID string, requested RepositoryRequest) (repositoryAccess, error) {
	if cfg.AgentID == "" || sessionID == "" {
		return repositoryAccess{}, fmt.Errorf("agent or session ID is missing")
	}
	id, err := uuid.Parse(requested.ID)
	if err != nil || id.String() != requested.ID || requested.Ref == "" {
		return repositoryAccess{}, fmt.Errorf("repository ID or ref is invalid")
	}
	if err := validBranch(ctx, requested.Ref); err != nil {
		return repositoryAccess{}, err
	}
	content, err := json.Marshal(map[string]string{
		"agent_session_id": sessionID,
		"repository_id":    requested.ID,
		"ref":              requested.Ref,
	})
	if err != nil {
		return repositoryAccess{}, err
	}
	scheme := "https"
	if isLoopbackHost(cfg.Host) {
		scheme = "http"
	}
	endpoint := scheme + "://" + cfg.Host + "/api/agents/" + url.PathEscape(cfg.AgentID) + "/repository-access/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(content))
	if err != nil {
		return repositoryAccess{}, fmt.Errorf("repository access request is invalid: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return repositoryAccess{}, fmt.Errorf("repository access request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return repositoryAccess{}, fmt.Errorf("repository access refused (HTTP %d); check your repository and ref permissions", resp.StatusCode)
	}
	var access repositoryAccess
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&access); err != nil {
		return repositoryAccess{}, fmt.Errorf("repository access answer is invalid: %w", err)
	}
	if !validRepositoryName(access.Repository) || access.Ref != requested.Ref || access.Token == "" {
		return repositoryAccess{}, fmt.Errorf("repository access answer is incomplete or does not match the selected ref")
	}
	return access, nil
}

func validRepositoryName(name string) bool {
	if !repositoryNamePattern.MatchString(name) {
		return false
	}
	owner, repo, _ := strings.Cut(name, "/")
	return owner != "." && owner != ".." && repo != "." && repo != ".."
}

func validBranch(ctx context.Context, ref string) error {
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("selected ref is invalid")
	}
	if err := exec.CommandContext(ctx, "git", "check-ref-format", "--branch", ref).Run(); err != nil {
		return fmt.Errorf("selected ref is invalid")
	}
	return nil
}

func prepareRepository(ctx context.Context, cfg Config, sessionID string, requested RepositoryRequest) (string, error) {
	access, err := requestRepositoryAccess(ctx, cfg, sessionID, requested)
	if err != nil {
		return "", err
	}
	root := filepath.Join(cfg.Workspace, "repositories")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create repository directory: %w", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("repository directory is not a regular directory")
	}
	dir := filepath.Join(root, requested.ID)
	remote := "https://github.com/" + access.Repository + ".git"
	env := append(os.Environ(),
		repositorySessionEnv+"="+sessionID,
		repositoryIDEnv+"="+requested.ID,
		repositoryRefEnv+"="+requested.Ref,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=/bin/false",
		"GIT_TRACE=0",
		"GIT_TRACE_CURL=0",
	)
	if info, err := os.Lstat(dir); os.IsNotExist(err) {
		stage, err := os.MkdirTemp(root, ".clone-")
		if err != nil {
			return "", fmt.Errorf("create clone directory: %w", err)
		}
		defer os.RemoveAll(stage)
		if err := runRepositoryGit(ctx, root, env, "clone", "-q", remote, stage); err != nil {
			return "", fmt.Errorf("clone repository failed; check repository access: %w", err)
		}
		if err := os.Rename(stage, dir); err != nil {
			return "", fmt.Errorf("place repository clone: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("read repository directory: %w", err)
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("repository checkout path is not a regular directory")
	}
	if err := verifyRepository(ctx, dir, remote, env); err != nil {
		return "", err
	}
	fetchRef := "refs/heads/" + access.Ref
	if pullRefPattern.MatchString(access.Ref) {
		fetchRef = "refs/" + access.Ref + "/head"
	}
	if err := runRepositoryGit(ctx, dir, env, "fetch", "-q", "--no-tags", "origin", fetchRef); err != nil {
		return "", fmt.Errorf("fetch selected ref failed; check repository access and ref: %w", err)
	}
	if err := runRepositoryGit(ctx, dir, env, "checkout", "-q", "--no-overwrite-ignore", "--detach", "FETCH_HEAD"); err != nil {
		return "", fmt.Errorf("checkout selected ref failed: %w", err)
	}
	for _, args := range [][]string{
		{"config", "--local", "--replace-all", repositorySessionConfig, sessionID},
		{"config", "--local", "--replace-all", repositoryIDConfig, requested.ID},
		{"config", "--local", "--replace-all", repositoryRefConfig, requested.Ref},
		{"config", "--local", "--replace-all", "credential.helper", ""},
		{"config", "--local", "--add", "credential.helper", repositoryCredentialHelper},
		{"config", "--local", "credential.https://github.com.useHttpPath", "true"},
	} {
		if err := runRepositoryGit(ctx, dir, env, args...); err != nil {
			return "", fmt.Errorf("configure repository credential helper: %w", err)
		}
	}
	return dir, nil
}

func verifyRepository(ctx context.Context, dir, remote string, env []string) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir, cmd.Env = dir, env
	root, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("repository directory is not a Git worktree")
	}
	actual, err := filepath.EvalSymlinks(strings.TrimSpace(string(root)))
	if err != nil {
		return fmt.Errorf("resolve repository worktree: %w", err)
	}
	selected, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("resolve repository directory: %w", err)
	}
	if actual != selected {
		return fmt.Errorf("repository directory is not its own Git worktree")
	}
	cmd = exec.CommandContext(ctx, "git", "config", "--local", "--get", "remote.origin.url")
	cmd.Dir, cmd.Env = dir, env
	got, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(got)) != remote {
		return fmt.Errorf("repository directory has a different origin")
	}
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir, cmd.Env = dir, env
	status, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("read repository status failed")
	}
	if len(status) != 0 {
		return fmt.Errorf("repository has local changes; save them before selecting another ref")
	}
	return nil
}

func runRepositoryGit(ctx context.Context, dir string, env []string, args ...string) error {
	options := []string{"-c", "credential.helper=", "-c", "credential.helper=" + repositoryCredentialHelper, "-c", "credential.https://github.com.useHttpPath=true"}
	cmd := exec.CommandContext(ctx, "git", append(options, args...)...)
	cmd.Dir, cmd.Env = dir, env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", args[0], err)
	}
	return nil
}
