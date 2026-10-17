# Spekk CLI 1.37.0 - Search Specs, Docs, and Source Documents From One Index

Before this release, the index held spec metadata only: no body and no full-text search. To find a fact in spec prose, a doc, or a document that a spec is written from, an agent used `grep` and read whole files. Source documents (transcripts, manuals, PDFs, email) had no place in the repository that a tool read.

This release adds one full-text index and four commands. See [Search Specs, Docs, and Source Documents](../source-documents.md). Issue #204.

## One index, four kinds

`.spekk/index.db` now holds a passage for each heading section of each spec (`spec`), assertion (`assertion`), doc in the repository (`doc`: `docs/**/*.md`, `docs/**/*.txt`, and `*.md` by default, or the globs in `.spekk/docs-paths`), and source document (`raw`). A section longer than 80 lines is cut into windows of 40 lines. The index is SQLite FTS5 with BM25 ranking. There is no vector store and no new service.

## `spekk docs search` and `spekk docs show`

`search` prints the best passage of each document with its line range, and filters on `--kind`, `--type`, `--tag`, and `--since`. Each word matches as written, so `snake_case`, `dotted.name`, and `kebab-case` match exactly, and a `camelCase` identifier also matches its parts. The output is at most 4,000 bytes, text or `--json`. `show <id> --lines a:b` prints numbered lines, at most 200 for each call. Text of kinds `doc` and `raw` is labeled as untrusted data, and control characters are removed.

## `spekk docs add` and `spekk docs status`

`add` copies a document into `raw/`, converts it to plaintext in `formatted/<id>.txt` when it must (HTML, email, and XLSX in the binary; PDF with `pdftotext`; DOCX with `pandoc`), and writes the stub `formatted/<id>.md` with a `sha256`. The summary is required for a complete stub: the command asks for it and checks its length. `add` refuses a file that holds a private key or an API token, and keeps a converted file larger than 5 MB out of git. `status` lists documents that are stale (the file changed), have no summary, have no file, or have no stub.

## Sources in specs

A spec or an assertion can cite documents: `sources: [doc-id]`. `spekk validate` warns when an id is unknown, and warns about each source document that needs work. The exit code does not change.

## Upgrade notes

- The index schema is now version 6. The first command after the upgrade rebuilds the index with no manual step.
- The index now follows the documents at query time. It keeps the size, mtime, and sha256 of each document file, indexes again only the documents of added, edited, and deleted files, and does not index a file again when only its mtime changed. When more than 10 files changed, a search answers at once from the current index, says so, and a background `spekk index` updates it. The index uses SQLite WAL, so parallel searches do not block or fail. A spec edit does not index the documents again.
- The coach, builder, and observer prompts now use `spekk docs search` for spec prose and documents.
