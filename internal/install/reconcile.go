package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	Hash     string // hash recorded in the stamp
	Pristine bool   // the on-disk body still agrees with the stamp
}

// scanOwned scans dirs for spekk-managed files and returns the owned set. A file
// with no stamp (user content) is ignored. A directory that does not exist
// contributes nothing and is not an error.
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
			if e.IsDir() {
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
			_, h, _ := ParseStamp(content)
			seen[path] = true
			owned = append(owned, OwnedFile{Path: path, Hash: h, Pristine: pristine})
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

// backupFile copies path to a ".bak" file that does not already exist, so it
// never overwrites an earlier backup (which could be the user's own file).
func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	bak := path + ".bak"
	for i := 1; ; i++ {
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			break
		}
		bak = fmt.Sprintf("%s.bak.%d", path, i)
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
		stamped := StampContent(desired[path])
		if existing, err := os.ReadFile(path); err == nil {
			if bytes.Equal(existing, stamped) {
				continue // already correct: no-op (idempotent)
			}
			managed, pristine := isPristine(existing)
			switch {
			case managed && pristine:
				// A managed file from another version: overwrite with the new content.
			case !managed && looksLikeSpekkShim(existing):
				// An old, unstamped spekk file from before the reconciler. It is
				// ours, not the user's. Back it up, then update it.
				if err := backupFile(path); err != nil {
					return res, fmt.Errorf("backing up %s: %w", path, err)
				}
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("%s was an unstamped spekk file; wrote %s.bak and updated it", path, path))
			default:
				// A hand-edited managed file, or genuine user content: do not clobber.
				if err := backupFile(path); err != nil {
					return res, fmt.Errorf("backing up %s: %w", path, err)
				}
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("%s was changed by hand; wrote %s.bak and left the file as is", path, path))
				continue
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

// CheckStale scans every supported target and scope for owned files that are not
// in that target's desired set. It reads files only; it changes nothing. It
// returns one line per stale file, each naming the file and the install command
// that migrates it.
func CheckStale(home, cwd string) ([]string, error) {
	var stale []string
	names := ValidTargets()
	for _, name := range names {
		t := targets[name]
		scopes := []bool{false} // global
		if t.projectDir != "" && cwd != "" {
			scopes = append(scopes, true) // project
		}
		for _, project := range scopes {
			dirs := t.managedDirs(project, home, cwd)
			want := t.desiredPaths(project, home, cwd)
			wantSet := map[string]bool{}
			for _, p := range want {
				wantSet[p] = true
			}
			owned, err := scanOwned(dirs)
			if err != nil {
				return nil, err
			}
			for _, o := range owned {
				if wantSet[o.Path] {
					continue
				}
				cmd := "spekk install --target " + name
				if project {
					cmd += " --project"
				}
				stale = append(stale, fmt.Sprintf("%s (run: %s)", o.Path, cmd))
			}
		}
	}
	sort.Strings(stale)
	return stale, nil
}

// looksLikeSpekkShim reports whether content is one of spekk's agent shims. The
// shim body always tells the session to run `spekk prompt <role>` and names the
// spekk agent, so a plain user file does not match by accident.
func looksLikeSpekkShim(content []byte) bool {
	return bytes.Contains(content, []byte("spekk prompt ")) &&
		bytes.Contains(content, []byte("You are the spekk"))
}

// migrateLegacy removes an old coach or builder agent shim that a role no longer
// uses. It handles only an unstamped file from a version before the reconciler;
// the reconciler prunes a stamped shim. It backs up the file first, because it
// cannot prove the file is unchanged. A file at a legacy path that is not a spekk
// shim (a file the user wrote) is left alone.
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
		if !looksLikeSpekkShim(content) {
			continue // not a spekk shim: leave the user's file
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
