package observer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
	"github.com/spekk-ai/spekk-cli/internal/index"
	"github.com/spekk-ai/spekk-cli/internal/observation"
)

// LogFileName is the repo-root-relative path of the announce failure log.
// The log records failures only: a successful run — including the valid
// "nothing to announce" outcome — never writes to it, so a non-empty log
// always means something needs attention. It is local operational telemetry
// (gitignored), not repo state: repo state lives in branches + frontmatter.
const LogFileName = ".spekk/observer-conversation.log"

// AnnounceOptions carries the injectable dependencies of the announce flow,
// so tests can pin the clock and the environment.
type AnnounceOptions struct {
	Dir      string              // working directory (any path inside the repo); "" = process cwd
	SpecsDir string              // specs directory for the index freshness gate; "" = <repoRoot>/specs
	Getenv   func(string) string // nil = os.Getenv
	Now      func() time.Time    // nil = time.Now
	Stdout   io.Writer           // nil = io.Discard
	Stderr   io.Writer           // nil = io.Discard
}

// batchCap is the most findings one announce message may carry. Findings
// past the cap stay unannounced and wait for the next run, so a backlog
// drains a few findings per run without a flood.
const batchCap = 3

// Announce runs one announce invocation and returns the process exit code.
//
// The invocation, in order: git fetch (the only remote read); refresh the
// index; select the unannounced open observations (high/medium only, high
// first, oldest first within severity, on branches visible on origin, at
// most batchCap of them); deliver them as ONE conversation-open request;
// then — only after delivery succeeded — commit the announced: flip to each
// observer branch and push (the only remote writes). At most ONE message is
// sent per invocation.
//
// Every failure appends a line to .spekk/observer-conversation.log and
// returns non-zero, leaving the frontmatter without the flip so the next run
// retries. The ordering is deliberate: deliver first, then mark. The worst
// case is a duplicate mention of a finding whose flip failed after delivery,
// never a finding silently marked announced that no human saw. No code path
// prompts for input or blocks on a TTY.
func Announce(opts AnnounceOptions) int {
	if opts.Getenv == nil {
		opts.Getenv = os.Getenv
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Dir == "" {
		opts.Dir, _ = os.Getwd()
	}

	root, err := repoRoot(opts.Dir)
	if err != nil {
		// Without a repo there is no canonical log location; fall back to
		// the working directory so the failure still leaves a trace.
		root = opts.Dir
	}
	fail := func(slug string, failErr error) int {
		logAnnounceFailure(root, opts.Now(), slug, failErr, opts.Stderr)
		return 1
	}
	if err != nil {
		return fail("", err)
	}

	g := gitRunner{dir: root}

	// 1. Fetch — the flow's only remote read.
	if err := g.fetch(); err != nil {
		return fail("", fmt.Errorf("fetch failed: %w", err))
	}

	// 2. Refresh the index (the fingerprint gate sees any newly fetched
	// observer branches).
	specsDir := opts.SpecsDir
	if specsDir == "" {
		specsDir = filepath.Join(root, "specs")
	}
	dbPath := index.DBPath(root)
	if _, err := index.EnsureFresh(specsDir, dbPath); err != nil {
		return fail("", fmt.Errorf("index refresh failed: %w", err))
	}

	// 3. Deterministic selection from the index.
	rows, err := loadCandidates(dbPath)
	if err != nil {
		return fail("", fmt.Errorf("candidate query failed: %w", err))
	}
	candidates := SelectCandidates(rows)

	// Only branches visible on origin are announce-eligible: skip candidates
	// whose branch exists only locally (pushing the branch is the scan's
	// job). Gather origin-visible candidates in the deterministic order, up
	// to batchCap — they form the one message of this run.
	type readyItem struct {
		cand      Candidate
		originRef string
		content   string
	}
	var ready []readyItem
	for i := range candidates {
		if len(ready) == batchCap {
			fmt.Fprintf(opts.Stdout, "%d more findings wait for the next run\n", len(candidates)-i)
			break
		}
		branch := observation.BranchName(candidates[i].Slug)
		ref, onOrigin, err := g.originRef(branch)
		if err != nil {
			return fail(candidates[i].Slug, fmt.Errorf("origin visibility check failed: %w", err))
		}
		if !onOrigin {
			fmt.Fprintf(opts.Stderr, "skipping %s: branch %s is not on origin\n", candidates[i].Slug, branch)
			continue
		}

		// Load the observation body from the origin ref for the evidence
		// summary, and re-run the evidence gate against the file itself.
		content, err := g.fileAt(ref, candidates[i].File)
		if err != nil {
			return fail(candidates[i].Slug, fmt.Errorf("cannot read observation at %s: %w", ref, err))
		}
		obs, err := observation.Parse(candidates[i].File, content)
		if err != nil {
			return fail(candidates[i].Slug, fmt.Errorf("observation invalid at announce time: %w", err))
		}
		if obs.Announced != "" {
			// The index lagged behind a concurrent flip; skip it.
			continue
		}
		candidates[i].Body = obs.Body
		ready = append(ready, readyItem{cand: candidates[i], originRef: ref, content: content})
	}
	if len(ready) == 0 {
		// Silence is a valid, successful outcome — no announcement, no log.
		fmt.Fprintln(opts.Stdout, "nothing to announce")
		return 0
	}

	// 4. Sandbox constraint: the spool directory comes from the session
	// environment. Outside a sandbox session announce cannot deliver and
	// must fail loudly rather than pretend to announce.
	spoolDir := opts.Getenv(conversation.SpoolEnvVar)
	if spoolDir == "" {
		return fail(ready[0].cand.Slug, fmt.Errorf("%s is not set; \"spekk observer announce\" must run inside a sandbox session", conversation.SpoolEnvVar))
	}

	// 5. Deliver: the batch is ONE conversation-open request.
	cands := make([]Candidate, len(ready))
	for i, r := range ready {
		cands[i] = r.cand
	}
	req := composeBatch(cands)
	if err := conversation.WriteRequest(spoolDir, req); err != nil {
		return fail(ready[0].cand.Slug, fmt.Errorf("conversation request failed: %w", err))
	}
	for _, r := range ready {
		fmt.Fprintf(opts.Stdout, "announced %s (%s)\n", r.cand.Slug, r.cand.Severity)
	}

	// 6. Delivery succeeded: record it as the announced: frontmatter flip on
	// each observer branch, and push. A failure past this point still logs
	// and exits non-zero, but the conversation is already open — the
	// documented worst case is a duplicate mention of the unflipped findings
	// on the next run.
	ts := opts.Now().UTC().Format("2006-01-02T15:04:05Z")
	for _, r := range ready {
		marked, err := observation.MarkAnnounced(r.content, ts)
		if err != nil {
			return fail(r.cand.Slug, fmt.Errorf("delivered, but cannot mark frontmatter: %w", err))
		}
		branch := observation.BranchName(r.cand.Slug)
		message := fmt.Sprintf("observer: mark %s announced", r.cand.Slug)
		// Capture the pre-flip tip before the push updates the
		// remote-tracking ref, so the local fast-forward check compares
		// against the right sha.
		parent, parentErr := g.run(nil, "rev-parse", r.originRef)
		commit, err := g.commitFileChange(r.originRef, r.cand.File, marked, message)
		if err != nil {
			return fail(r.cand.Slug, fmt.Errorf("delivered, but flip commit failed: %w", err))
		}
		if err := g.pushCommit(commit, branch); err != nil {
			return fail(r.cand.Slug, fmt.Errorf("delivered, but flip push failed: %w", err))
		}
		if parentErr == nil {
			g.fastForwardLocal(branch, commit, parent)
		}
		fmt.Fprintf(opts.Stdout, "marked %s announced (%s)\n", r.cand.Slug, ts)
	}
	return 0
}

// logAnnounceFailure appends one line to the failure log (creating it if
// absent, never truncating) and mirrors the error to stderr. The line
// carries a timestamp, the observation slug when one was selected, and the
// error.
func logAnnounceFailure(root string, now time.Time, slug string, failErr error, stderr io.Writer) {
	fmt.Fprintf(stderr, "Error: %s\n", failErr)

	if slug == "" {
		slug = "-"
	}
	line := fmt.Sprintf("%s slug=%s error=%s\n",
		now.UTC().Format("2006-01-02T15:04:05Z"), slug,
		strings.ReplaceAll(failErr.Error(), "\n", " "))

	logPath := filepath.Join(root, filepath.FromSlash(LogFileName))
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		fmt.Fprintf(stderr, "warning: cannot create log directory: %s\n", err)
		return
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, "warning: cannot open failure log: %s\n", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		fmt.Fprintf(stderr, "warning: cannot append to failure log: %s\n", err)
	}
	ensureLogGitignored(root)
}

// ensureLogGitignored makes sure the failure log stays out of version
// control. Best effort: the log write must never fail because .gitignore is
// unwritable.
func ensureLogGitignored(root string) {
	gitignorePath := filepath.Join(root, ".gitignore")
	raw, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == LogFileName || trimmed == ".spekk/" || trimmed == ".spekk" {
			return
		}
	}
	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	prefix := ""
	if len(raw) > 0 && !strings.HasSuffix(string(raw), "\n") {
		prefix = "\n"
	}
	_, _ = f.WriteString(prefix + LogFileName + "\n")
}
