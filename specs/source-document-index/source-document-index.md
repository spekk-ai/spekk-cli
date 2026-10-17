---
id: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
---

# Source Document Index - Search Specs, Docs, and Source Documents From One Index

Issue: #204.

## Problem

An agent that needs a fact from a document reads files with `grep` and `cat`. That costs tokens and time, and it scales badly: a long manual or transcript is read in full to find 40 lines. Spekk indexes spec metadata in `.spekk/index.db` (`specs`, `assertions`, no body, no full-text search), so it cannot help. The documents a spec is written from (transcripts, manuals, PDFs, email) have no place in the repository that a tool reads. Most other prose (`docs/`, ADRs, READMEs) is in the repository already, but nothing indexes it.

Our evals found that a CLI that queries metadata and an index is faster and uses fewer tokens than grep over files. This spec puts the bodies of specs and assertions, the repository's own docs, and a new corpus of source documents into one full-text index, and gives the agent a small command surface with a strict output budget.

## Design

### Four kinds, one index

Each indexed document has a `kind`:

| kind | What it is | Where it lives | Id |
|---|---|---|---|
| `spec` | A parent spec file | `specs/<id>/<id>.md` | the spec id |
| `assertion` | An assertion file | `specs/<id>/assertions/*.md` | the assertion id |
| `doc` | Prose that is already in the repository | in place (default `docs/**/*.md`, `docs/**/*.txt`, `*.md`) | kebab-case of the path without the extension (`docs/concepts.md` -> `docs-concepts`) |
| `raw` | A source document: evidence, not ours to edit | `raw/`, with one stub in `formatted/` | the stub file name (kebab-case of the path under `raw/` without the extension) |

A spec is a claim about what we will build. A source document is evidence. They share the index so that one search reaches both, and they stay apart on disk: nothing in `raw/` or `formatted/` is ever parsed as a spec, and `kind` keeps them apart in every result.

### Layout of the source corpus

- `raw/` holds each source document as it arrived. It is the source of truth.
- `formatted/<id>.md` is the stub: YAML frontmatter and no body. Fields: `title`, `type`, `created`, `source` (where the document came from), `path` (the file in `raw/`), `text` (the derived plaintext, only for a converted format), `sha256` (of the raw file when the stub was written), `size`, `tags`, and the mandatory `summary`.
- `formatted/<id>.txt` is the derived plaintext of a converted format (PDF, DOCX, XLSX, HTML, email). It is committed. The index reads plaintext only, so a rebuild needs no converter and the invariant of `internal/index` holds: everything in `.spekk/index.db` is rebuildable from plaintext in the repository.

### Documents that already live in the repository

`docs/`, READMEs, ADRs, and meeting notes stay where they are. They need no stub and no move. Optional frontmatter on the file (`title`, `summary`, `type`, `tags`, `created`) is read when present. `.spekk/docs-paths` changes the set: one glob for each line, `!` before a glob excludes, `#` starts a comment. With no file, the defaults apply. `specs/`, `raw/`, `formatted/`, `.spekk/`, and `.git/` are never in-place documents.

### Passages, not whole documents

BM25 over a whole long document ranks badly, and the second read is then the full file. The index holds passages: one passage for each heading section (ATX `#` headings and setext `===`/`---` underlines), and a section longer than 80 lines is cut into windows of 40 lines. A result gives the document, the best passage, and its line range in the plaintext file, so the agent reads 40 lines instead of 4,000.

### Command surface and output budget

- `spekk docs add <path>...` ingests source documents (files or folders). It writes the stub deterministically. The summary is an agent step that the command asks for and validates (`--summary`, 40 to 600 characters). It is never a convention.
- `spekk docs search "<query>" [--kind] [--type] [--tag] [--since] [--limit] [--json]` returns ranked documents with title, summary, the best passage and its line range. The output has a hard cap of 4,000 bytes, on by default, with no flag to remove it.
- `spekk docs show <id|path> [--lines a:b]` prints numbered lines of the plaintext, at most 200 lines for each call.
- `spekk docs status` lists documents that need work (stale, no summary, no stub, raw file missing).
- `spekk query` reads the new tables (`documents`, `document_tags`, `passages`) read-only, as today.

Search is on demand and never injected into a prompt. The agent calls it the way it reads a file.

### Tokenizer

SQLite FTS5 with `porter unicode61 remove_diacritics 2`. Code identifiers must match precisely: each query term is quoted, so `snake_case`, `dotted.name`, and `kebab-case` become exact phrases, and a `camelCase` identifier in a document is also indexed as its parts, so `parse specs` finds `parseAllSpecs`. Terms are joined with OR and ranked with BM25, so a document that matches more terms ranks first.

### Formats

| Format | Conversion | Why |
|---|---|---|
| Markdown, plain text | none | already plaintext |
| HTML | in the binary (`golang.org/x/net/html`) | common, and a small pure-Go parser exists |
| Email (`.eml`) | in the binary (`net/mail`, `mime/multipart`) | the standard library reads it, and no common converter exists |
| XLSX | in the binary (`archive/zip`, `encoding/xml`) | pandoc does not read XLSX, LibreOffice is too large to require, and the format is XML in a zip |
| PDF | `pdftotext` on `PATH` (poppler) | no good pure-Go PDF text extractor |
| DOCX | `pandoc` on `PATH` | pandoc keeps headings, so passages follow sections |

A missing converter stops the add of that file with a message that names the tool and how to install it. The recommendation was an external converter for XLSX and email too; the table gives the reason for each change.

### Binaries in git

Text goes in git. A converted file of more than 5 MB becomes a pointer: the stub keeps `sha256`, `size`, and `source`, the raw file is listed in `raw/.gitignore`, and the derived text is committed. The index then works on every clone, and a repository never grows by accident.

### Freshness

The index is checked at query time, and a query is never slow because of it. `EnsureFresh` stays the one freshness gate.

- **Spec part:** today's rule. A file or directory under `specs/` newer than `index.db`, or changed observation refs, rebuilds the spec part (about 0.2 s).
- **Document part, cheap dirty check:** the index holds a manifest (`doc_files`: path, size, mtime, sha256 of each file the corpus reads). A stat walk is compared with it. No file is read when nothing changed.
- **No false positives:** a file whose size or mtime changed is hashed. Only a file whose sha256 changed is re-indexed; a `touch` or a `git checkout` with the same content costs one hash and a manifest update.
- **Incremental:** only the documents of changed, added, and deleted files are deleted and inserted again (a changed stub, raw file, or derived text maps to its raw document). A change to `.spekk/docs-paths` or `raw/.gitignore`, a duplicate id, a schema change, and `spekk index --force` rebuild the whole document part.
- **Inline or background:** a small change set is applied inline, and the query answers from a fresh index. A large one (a fresh clone with an old index, a branch switch) does not block: the query answers from the current index at once, says so (`"stale": {"changed": N, "reindexing": true}` in JSON, one line in text), and starts one background `spekk index` that holds `.spekk/index.lock`. The index uses SQLite WAL and a busy timeout, so a reader is never blocked and a parallel query never fails.

Each stub keeps the `sha256` of its raw file; when the raw file changes, the stub is stale. `spekk docs status` lists it, `spekk validate` warns, and search marks the result `stale`. To add the document again clears the summary when the content changed, so the agent must write a new one.

### Untrusted content and sensitive documents

Source documents are someone else's text. Search and show output for kinds `raw` and `doc` is labeled as untrusted data, and terminal control characters are removed from all document text (the same rule as #147, applied to tool output). The markers around the text of `show` hold a random value for each call, so a document cannot forge the end marker. A document is hostile input in other ways too: a stub path that leaves `raw/` or `formatted/`, a symlink that leaves the repository, an XLSX cell reference or part that is too large, and a converter that does not stop are each refused. `spekk docs add` scans the extracted text for secrets (private keys, cloud and API tokens) and refuses the file unless `--allow-secrets` is given, because git history is permanent. The exclude globs in `.spekk/docs-paths` keep a file out of the index; they do not keep it out of git. A document that must not enter git does not belong in `raw/`.

### Links from specs to sources

An assertion or spec can cite source documents in frontmatter: `sources: [doc-id, ...]`. `spekk validate` warns (never fails) when an id is unknown. The observer prompt tells observers to read the cited documents, which is the evidence half of the spec/evidence split.

## Rejected

- **Vector search and embeddings.** Anthropic dropped vector search from Claude Code in favor of agentic grep. An Amazon Science AAAI-2026 paper measured agentic keyword search at about 94.5% of RAG faithfulness with no vector store. BM25 beats dense retrieval on precise terminology, and this corpus is mostly precise terminology: field names, model names, standard codes. An embedding model is also a new dependency and a new service.
- **A custom FTS5 tokenizer.** The pure-Go SQLite driver cannot register one without cgo. Quoted query terms and an extra column of identifier parts give the same matches.
- **A YAML config file.** The repository has no YAML library, and a list of globs needs none.
- **Auto-injection of search results into prompts.**
- **A separate database or service.** The existing `index.db` holds it.
- **A `--max-bytes` flag.** The cap is the product. A caller that needs more makes a second, narrower call.

## Out of scope

Vector search, a spekk-app UI for documents, Google Drive sync, OCR of scanned PDFs, PPTX.

## Acceptance

The claim rests on an eval, not on unit tests: about 20 fixed questions over a public corpus of a few hundred documents, answered by a headless agent once with only grep and read, and once with `spekk docs`. Tokens, wall time, tool calls, and correctness against a key written first. The eval suite lives outside this repository. The pull request quotes its numbers.
