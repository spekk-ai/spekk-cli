---
id: spec-passages-incremental
parent: source-document-index
created: 2026-10-17T20:30:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: index-holds-full-text
---

# A Spec Edit Re-Indexes Only the Passages of That Spec

## Success criteria

- The index has a manifest `spec_files (path, size, mtime_ns, sha256)` with one row for each regular `.md` file under `specs/`, as `doc_files` does for the document part. Only a file whose size or mtime differs from its row is read and hashed.
- When a file under `specs/` is newer than `index.db` but no `.md` file was added, deleted, or changed its content (a `touch`), the manifest rows are updated, the mtime of `index.db` is set to the start of the check, and the spec part is not rebuilt. The next command does not check again.
- When a `.md` file changed its content, the spec part is rebuilt: the spec and assertion rows (`specs`, `assertions`, `depends_on`, `frontmatter_fields`) are written again as on `main`. The `documents`, `document_tags`, and `passages` rows of a spec or assertion are kept when its file has the same path and sha256 as its `documents` row. One whose file changed, moved, or is new is deleted and inserted again with its passages, and one that is gone is deleted. So an edit of one assertion file writes the passages of one document.
- A spec tree that does not parse still fails each command until it is fixed: the check writes nothing when it finds a change.
- After an edit, `spekk docs search` finds the new text of the edited assertion and does not find its old text. The spec and assertion rows of `documents` and `passages` equal the rows of `spekk index --force`.
- On a tree of more than 1,000 spec files, the next `spekk` command after one spec edit takes at most 300 ms, and after one touch at most 250 ms, on a 2-vCPU machine (measured by hand, not in a test).

**Tests:** `internal/index/documents_test.go` `TestSpecEditIndexesOnlyThatSpec`: a touch alone does not rebuild and leaves the index fresh; after a touch of one assertion and an edit of another, the passages of the touched assertion keep their rowids, the edit is found, its old text is not, and the rows equal a forced rebuild.
