package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The managed marker is a single trailing line on every file that spekk
// install writes. A trailing marker works for every target, including codex,
// whose files carry no frontmatter. The marker records a hash of the body, so
// the reconciler can tell a pristine managed file from one the user changed.
const (
	managedMarkerPrefix = "<!-- spekk:managed hash="
	managedMarkerSuffix = " -->"
)

// bodyHash returns the lowercase hex SHA-256 of body.
func bodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// StampContent appends the managed marker to body. The marker holds the hash of
// body. StampContent adds exactly "\n" + marker + "\n", so ParseStamp recovers
// the original body byte-for-byte.
func StampContent(body []byte) []byte {
	marker := managedMarkerPrefix + bodyHash(body) + managedMarkerSuffix
	out := make([]byte, 0, len(body)+len(marker)+2)
	out = append(out, body...)
	out = append(out, '\n')
	out = append(out, marker...)
	out = append(out, '\n')
	return out
}

// ParseStamp reads a stamped file. For a stamped file it returns the body
// without the marker, the hash from the marker, and managed=true. For a file
// with no marker it returns the original content, "", and managed=false.
func ParseStamp(content []byte) (body []byte, hash string, managed bool) {
	s := content
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	idx := bytes.LastIndexByte(s, '\n')
	if idx < 0 {
		return content, "", false
	}
	last := string(s[idx+1:])
	if !strings.HasPrefix(last, managedMarkerPrefix) || !strings.HasSuffix(last, managedMarkerSuffix) {
		return content, "", false
	}
	h := strings.TrimSuffix(strings.TrimPrefix(last, managedMarkerPrefix), managedMarkerSuffix)
	if len(h) != 64 || strings.TrimLeft(h, "0123456789abcdef") != "" {
		return content, "", false
	}
	return s[:idx], h, true
}

// isPristine reports whether stamped content is a spekk-managed file whose body
// still agrees with the hash in its stamp.
func isPristine(content []byte) (managed, pristine bool) {
	body, h, managed := ParseStamp(content)
	if !managed {
		return false, false
	}
	return true, bodyHash(body) == h
}

// OwnedFile is one spekk-managed file found by a scan.
type OwnedFile struct {
	Path     string
	Pristine bool // the on-disk body still agrees with the stamp
}

// backupSuffix is the extension backupFile adds to the file it preserves.
const backupSuffix = ".bak"

// isBackupName reports whether name is a backup that backupFile wrote: either
// "<name>.bak" or the "<name>.bak.<n>" form it falls back to.
func isBackupName(name string) bool {
	return strings.HasSuffix(name, backupSuffix) || strings.Contains(name, backupSuffix+".")
}

// linkTarget returns what path points to when path is a symlink, and "" when it
// is a regular file or is not there.
//
// A managed path that is a symlink has a second owner: the tool that made the
// link, usually a dotfiles manager. spekk does not write through it and does not
// remove it. The far end is not a path spekk owns, and only the user can say
// which of the two tools should own the path. So spekk reports the conflict and
// leaves the path alone.
func linkTarget(path string) (string, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspecting %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return "", nil
	}
	dest, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("reading the link %s: %w", path, err)
	}
	return dest, nil
}

// symlinkWarning is the one wording for a managed path spekk left alone because
// another tool owns it through a link.
func symlinkWarning(path, dest string) string {
	return fmt.Sprintf("%s is a symlink to %s; spekk left it alone — decide which tool owns this path", path, dest)
}

// scanOwned scans dirs for spekk-managed files and returns the owned set. A file
// with no stamp (user content) is ignored. A directory that does not exist
// contributes nothing and is not an error.
//
// A backup spekk wrote is not owned. The backup of a managed file the user
// edited carries the old stamp, so a scan would otherwise call it a managed file
// that the current layout does not write: every install would back the backup up
// again, and every "spekk update" would report it as stale with an install
// command that only makes more copies.
//
// A symlink is not owned either, so the prune half never removes a link that
// another tool made.
func scanOwned(dirs []string) ([]OwnedFile, error) {
	seen := map[string]bool{}
	var owned []OwnedFile
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("scanning %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || isBackupName(e.Name()) || e.Type()&os.ModeSymlink != 0 {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if seen[path] {
				continue
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", path, err)
			}
			managed, pristine := isPristine(content)
			if !managed {
				continue
			}
			seen[path] = true
			owned = append(owned, OwnedFile{Path: path, Pristine: pristine})
		}
	}
	sort.Slice(owned, func(i, j int) bool { return owned[i].Path < owned[j].Path })
	return owned, nil
}

// Result records what a reconcile did.
type Result struct {
	Written  []string
	Removed  []string
	Warnings []string
}

// backupFile copies path to a backup file that does not already exist, so it
// never overwrites an earlier backup (which could be the user's own file).
func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	bak := path + backupSuffix
	for i := 1; ; i++ {
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			break
		}
		bak = fmt.Sprintf("%s%s.%d", path, backupSuffix, i)
	}
	return os.WriteFile(bak, data, 0o644)
}

// reconcile drives the managed files in dirs to the desired set. desired maps a
// destination path to its unstamped body. reconcile stamps and writes each
// desired file, removes owned files that are not desired, and never changes a
// file the user edited (it makes a .bak backup and records a warning instead).
func reconcile(desired map[string][]byte, dirs []string) (Result, error) {
	var res Result

	owned, err := scanOwned(dirs)
	if err != nil {
		return res, err
	}

	// Write or update the desired files.
	paths := make([]string, 0, len(desired))
	for p := range desired {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		dest, err := linkTarget(path)
		if err != nil {
			return res, err
		}
		if dest != "" {
			res.Warnings = append(res.Warnings, symlinkWarning(path, dest))
			continue
		}
		stamped := StampContent(desired[path])
		if existing, err := os.ReadFile(path); err == nil {
			if bytes.Equal(existing, stamped) {
				continue // already correct: no-op (idempotent)
			}
			// The path is the statement of ownership: this is a path spekk writes,
			// so spekk brings it to the current content. The stamp decides only
			// whether a backup is necessary first. A pristine managed file is
			// spekk's own content from another version and needs no backup;
			// anything else — a managed file the user edited, or a file with no
			// stamp — is kept in a .bak before the update.
			if managed, pristine := isPristine(existing); !managed || !pristine {
				if err := backupFile(path); err != nil {
					return res, fmt.Errorf("backing up %s: %w", path, err)
				}
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("%s did not match the content spekk installed; wrote %s.bak and updated it", path, path))
			}
		} else if !os.IsNotExist(err) {
			return res, fmt.Errorf("reading %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return res, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, stamped, 0o644); err != nil {
			return res, fmt.Errorf("writing %s: %w", path, err)
		}
		res.Written = append(res.Written, path)
	}

	// Prune owned files that are not desired.
	for _, o := range owned {
		if _, ok := desired[o.Path]; ok {
			continue
		}
		if !o.Pristine {
			if err := backupFile(o.Path); err != nil {
				return res, fmt.Errorf("backing up %s: %w", o.Path, err)
			}
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s was changed by hand; wrote %s.bak and did not remove the file", o.Path, o.Path))
			continue
		}
		if err := os.Remove(o.Path); err != nil {
			return res, fmt.Errorf("removing %s: %w", o.Path, err)
		}
		res.Removed = append(res.Removed, o.Path)
	}

	return res, nil
}

// StaleReason says why an installed file does not match this binary.
type StaleReason int

const (
	// StaleOldLayout: spekk owns the file, but the current layout does not write it.
	StaleOldLayout StaleReason = iota
	// StaleOutOfDate: the file sits at a path spekk writes, but its content is not
	// what this binary installs.
	StaleOutOfDate
	// StaleSymlink: the path spekk writes is a symlink, so a second tool owns it.
	// An install does not fix this one; the user has to choose an owner.
	StaleSymlink
)

var staleReasonText = map[StaleReason]string{
	StaleOldLayout: "is from an old layout",
	StaleOutOfDate: "is out of date",
	StaleSymlink:   "is a symlink to another tool's file",
}

func (r StaleReason) String() string {
	if s, ok := staleReasonText[r]; ok {
		return s
	}
	return fmt.Sprintf("StaleReason(%d)", int(r))
}

// StaleFile is one installed file that does not match this binary.
type StaleFile struct {
	Path       string
	Reason     StaleReason
	Target     string // the install target that claims the path
	Project    bool   // the path belongs to a project-scope install
	LinkTarget string // what the path points to; set only for StaleSymlink
}

// Remedy returns what the user must do about this file. An install fixes a stale
// layout or stale content, but it cannot fix a path a second tool owns.
func (s StaleFile) Remedy() string {
	if s.Reason == StaleSymlink {
		return "decide which tool owns this path"
	}
	return "run: " + installScope{name: s.Target, project: s.Project}.installCommand()
}

// installScope is one target in one scope: everything a scan of the install
// locations needs to know.
type installScope struct {
	name    string
	target  target
	project bool
}

// installCommand returns the command that installs this target and scope.
func (s installScope) installCommand() string {
	cmd := "spekk install --target " + s.name
	if s.project {
		cmd += " --project"
	}
	return cmd
}

// eachInstallScope returns every target and scope a scan must visit. The global
// scope of a target comes before its project scope, so a caller that keeps the
// first result for a path prefers the global install command.
func eachInstallScope(cwd string) []installScope {
	var scopes []installScope
	for _, name := range ValidTargets() {
		t := targets[name]
		scopes = append(scopes, installScope{name: name, target: t})
		if t.projectDir != "" && cwd != "" {
			scopes = append(scopes, installScope{name: name, target: t, project: true})
		}
	}
	return scopes
}

// InstalledTargets returns the install command for every target and scope that
// has at least one spekk-managed file on disk. It reads files only. A caller uses
// it to name the installs a user actually has, instead of listing every target
// spekk supports.
func InstalledTargets(home, cwd string) ([]string, error) {
	seen := map[string]bool{}
	var cmds []string
	for _, s := range eachInstallScope(cwd) {
		dirs := s.target.managedDirs(s.project, home, cwd)
		// Two scopes can name the same directories when the working directory is
		// the home directory. Report the first (global) command only.
		fresh := false
		for _, d := range dirs {
			if !seen[d] {
				fresh = true
			}
		}
		if !fresh {
			continue
		}
		owned, err := scanOwned(dirs)
		if err != nil {
			return nil, err
		}
		if len(owned) == 0 {
			continue
		}
		for _, d := range dirs {
			seen[d] = true
		}
		cmds = append(cmds, s.installCommand())
	}
	sort.Strings(cmds)
	return cmds, nil
}

// CheckStale scans every supported target and scope for two conditions: an owned
// file that is not in that target's desired set (an old layout), and a file at a
// desired path whose content does not match what this binary installs (an
// out-of-date file). It reads files only; it changes nothing.
//
// It reports each path once. Two scopes can name the same path — a user whose
// working directory is their home directory has the same .claude directory in
// both — and one file with two contradictory fix commands helps nobody. The
// global scope is checked first, so the global command wins the tie.
func CheckStale(home, cwd string, skillFS fs.FS) ([]StaleFile, error) {
	seen := map[string]bool{}
	var stale []StaleFile
	add := func(f StaleFile) {
		if seen[f.Path] {
			return
		}
		seen[f.Path] = true
		stale = append(stale, f)
	}

	for _, s := range eachInstallScope(cwd) {
		desired, err := s.target.desiredFiles(s.project, home, cwd, skillFS)
		if err != nil {
			return nil, err
		}

		// An owned file the current layout no longer writes.
		owned, err := scanOwned(s.target.managedDirs(s.project, home, cwd))
		if err != nil {
			return nil, err
		}
		for _, o := range owned {
			if _, want := desired[o.Path]; want {
				continue
			}
			add(StaleFile{Path: o.Path, Reason: StaleOldLayout, Target: s.name, Project: s.project})
		}

		// A file that is installed at a desired path but is not current. A
		// desired path with no file there is not installed, so it is silent.
		for path, body := range desired {
			dest, err := linkTarget(path)
			if err != nil {
				return nil, err
			}
			if dest != "" {
				add(StaleFile{Path: path, Reason: StaleSymlink, Target: s.name, Project: s.project, LinkTarget: dest})
				continue
			}
			existing, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("reading %s: %w", path, err)
			}
			if bytes.Equal(existing, StampContent(body)) {
				continue
			}
			add(StaleFile{Path: path, Reason: StaleOutOfDate, Target: s.name, Project: s.project})
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Path < stale[j].Path })
	return stale, nil
}

// migrateLegacy removes an old coach or builder agent shim that a role no longer
// uses. The path is spekk's own: spekk wrote that spekk-namespaced name into the
// host's agent directory, so migrateLegacy removes the file whatever the body
// says. It backs up the file first, because it cannot prove the file is
// unchanged. It handles only an unstamped file from a version before the
// reconciler; the reconciler prunes a stamped shim.
//
// It skips a legacy path that is also a desired path — a host such as codex that
// keeps agents and skills in one directory, or a host with no skill path where
// the role stays an agent shim. The reconcile updates that file in place.
func migrateLegacy(paths []string, desired map[string][]byte) (Result, error) {
	var res Result
	for _, p := range paths {
		if _, ok := desired[p]; ok {
			continue // a desired path: the reconcile updates it in place
		}
		dest, err := linkTarget(p)
		if err != nil {
			return res, err
		}
		if dest != "" {
			res.Warnings = append(res.Warnings, symlinkWarning(p, dest))
			continue
		}
		content, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return res, fmt.Errorf("reading %s: %w", p, err)
		}
		if _, _, managed := ParseStamp(content); managed {
			continue // the reconciler owns and prunes a stamped file
		}
		if err := backupFile(p); err != nil {
			return res, fmt.Errorf("backing up %s: %w", p, err)
		}
		if err := os.Remove(p); err != nil {
			return res, fmt.Errorf("removing %s: %w", p, err)
		}
		res.Removed = append(res.Removed, p)
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s was a legacy agent shim; wrote %s.bak and removed it", p, p))
	}
	return res, nil
}
