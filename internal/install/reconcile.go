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

// symlinkWarning is the one wording for a managed path spekk left alone.
func symlinkWarning(path, dest string) string {
	return fmt.Sprintf("%s is a symlink to %s; spekk left it alone — decide which tool owns this path", path, dest)
}

// scanOwned scans dirs for the spekk-managed files in them.
//
//   - A stamped file is owned.
//   - A file with no stamp is the user's, and is ignored.
//   - A backup spekk wrote is not owned. It carries an old stamp, so counting it
//     would back it up again on every install.
//   - A symlink is not owned. Another tool made it.
//   - A directory that does not exist contributes nothing, and is not an error.
//   - A file spekk cannot read is not owned. Two of these directories hold the
//     user's own prompts beside spekk's files, so one file another program
//     locked down must not disable the whole scan.
//   - Only a regular file is read. A FIFO in one of these directories would
//     block the read forever.
func scanOwned(dirs []string) ([]OwnedFile, error) {
	seen := map[string]bool{}
	var owned []OwnedFile
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				continue
			}
			return nil, fmt.Errorf("scanning %s: %w", dir, err)
		}
		for _, e := range entries {
			if !e.Type().IsRegular() || isBackupName(e.Name()) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if seen[path] {
				continue
			}
			content, err := os.ReadFile(path)
			if err != nil {
				// The file went away, or it is not ours to read.
				if os.IsNotExist(err) || os.IsPermission(err) {
					continue
				}
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

// backupFile copies path to a backup file and returns the path it wrote or
// kept. It never overwrites an earlier backup, which could be the user's own
// file: it falls back to "<path>.bak.1", "<path>.bak.2", and so on. When a
// backup already holds the same bytes, it keeps that one and writes nothing, so
// repeated installs cannot pile up identical copies. The backup keeps the mode
// of the file it preserves, so a private file does not become readable.
func backupFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	bak := path + backupSuffix
	for i := 1; ; i++ {
		existing, err := os.ReadFile(bak)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", err
		}
		if bytes.Equal(existing, data) {
			return bak, nil // the same content is already preserved
		}
		bak = fmt.Sprintf("%s%s.%d", path, backupSuffix, i)
	}
	if err := os.WriteFile(bak, data, mode); err != nil {
		return "", err
	}
	return bak, nil
}

// dirKey resolves every symlink in dir so two spellings of one directory share
// a key. A user whose working directory reaches home through a symlink has one
// .claude directory under two names. dirKey returns dir unchanged when the path
// cannot be resolved, which is the case for a directory that is not there.
func dirKey(dir string) string {
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved
	}
	return dir
}

// pathKey is dirKey for the parent of path. It resolves a symlinked ancestor
// but never follows a symlink at path itself, so two managed paths that point
// at one file stay two separate reports.
func pathKey(path string) string {
	return filepath.Join(dirKey(filepath.Dir(path)), filepath.Base(path))
}

// reconcile drives the managed files in dirs to the desired set. desired maps a
// destination path to its unstamped body.
//
//   - A desired path belongs to spekk. reconcile stamps the body and writes it.
//   - It keeps a .bak of what it replaced, unless the file was a pristine
//     managed file.
//   - It removes an owned file that is not desired. An owned file the user
//     edited is backed up and left in place instead.
//   - It leaves a symlinked path alone. Another tool owns that one.
//   - Each backup and each path left alone is one warning in the Result.
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
			// A pristine managed file is spekk's own content from another
			// version. Every other file is kept in a .bak before the update.
			if managed, pristine := isPristine(existing); !managed || !pristine {
				bak, err := backupFile(path)
				if err != nil {
					return res, fmt.Errorf("backing up %s: %w", path, err)
				}
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("%s did not match the content spekk installed; wrote %s and updated it", path, bak))
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
			bak, err := backupFile(o.Path)
			if err != nil {
				return res, fmt.Errorf("backing up %s: %w", o.Path, err)
			}
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s was changed by hand; wrote %s and did not remove the file", o.Path, bak))
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
	// The user has to choose an owner.
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
// layout and stale content. It does not fix a path a second tool owns.
func (s StaleFile) Remedy() string {
	if s.Reason == StaleSymlink {
		return "decide which tool owns this path"
	}
	return "run: " + installScope{name: s.Target, project: s.Project}.installCommand()
}

// installScope is one target in one scope.
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
// scope of a target comes before its project scope.
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
// has at least one spekk-managed file on disk. It reads files only.
func InstalledTargets(home, cwd string) ([]string, error) {
	seen := map[string]bool{}
	var cmds []string
	for _, s := range eachInstallScope(cwd) {
		dirs := s.target.managedDirs(s.project, home, cwd)
		// Two scopes can name the same directories when the working directory is
		// the home directory. Report the first (global) command only. The key is
		// the resolved directory, so a cwd that reaches home through a symlink
		// is the same directory here too.
		fresh := false
		for _, d := range dirs {
			if !seen[dirKey(d)] {
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
			seen[dirKey(d)] = true
		}
		cmds = append(cmds, s.installCommand())
	}
	sort.Strings(cmds)
	return cmds, nil
}

// CheckStale reports every installed file that this binary does not match. It
// reads files only.
//
//   - An owned file the current layout does not write is an old layout.
//   - A file at a desired path whose content differs is out of date.
//   - A symlink at a desired path belongs to another tool.
//
// It reports each path once, sorted by path. The global scope is checked before
// the project scope, so the global command wins when two scopes name one path.
// One file reached by two spellings is one path: the key resolves a symlinked
// ancestor, which is what a working directory that reaches home through a link
// produces.
func CheckStale(home, cwd string, skillFS fs.FS) ([]StaleFile, error) {
	seen := map[string]bool{}
	var stale []StaleFile
	add := func(f StaleFile) {
		key := pathKey(f.Path)
		if seen[key] {
			return
		}
		seen[key] = true
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
// uses. The path belongs to spekk, so the body does not decide.
//
//   - It backs the file up first. It cannot prove the file is unchanged.
//   - It skips a stamped file. reconcile prunes that one.
//   - It skips a symlink. Another tool owns that one.
//   - It skips a legacy path that is also a desired path. reconcile updates that
//     file in place. Codex reaches this case: it keeps agents and skills in one
//     directory.
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
		bak, err := backupFile(p)
		if err != nil {
			return res, fmt.Errorf("backing up %s: %w", p, err)
		}
		if err := os.Remove(p); err != nil {
			return res, fmt.Errorf("removing %s: %w", p, err)
		}
		res.Removed = append(res.Removed, p)
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s was a legacy agent shim; wrote %s and removed it", p, bak))
	}
	return res, nil
}
