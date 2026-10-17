---
id: docs-add-converted-formats
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
status: not_started
branch: feature/source-document-index
depends-on: docs-add-text-formats
---

# `spekk docs add` Converts PDF, DOCX, XLSX, and Email to Committed Plaintext

## Success criteria

- `.pdf` is converted with `pdftotext -enc UTF-8 <file> -`. `.docx` is converted with `pandoc --to gfm --wrap none <file>`, so headings stay headings. Each converter is found on `PATH`.
- When a converter is missing, that file fails with: `cannot convert <file>: <tool> is not on PATH. Install <package> (apt: <apt package>, brew: <brew package>), or convert the file to text and add the text file.` Other files in the same call continue.
- `.xlsx` is converted in the binary: one `# <sheet name>` heading for each sheet, then one line for each row with the cell values joined by a tab. Shared strings and inline strings are resolved. Numbers are written as stored.
- `.eml` is converted in the binary: `From`, `To`, `Date`, and `Subject` lines, a blank line, then the `text/plain` part. With no `text/plain` part, the `text/html` part goes through the HTML conversion. Quoted-printable and base64 bodies are decoded. Attachments are listed by file name only.
- The derived text is written to `formatted/<id>.txt` and the stub has `text: formatted/<id>.txt`. The index reads this file and never the binary, so `spekk index --force` on a clone with no converter gives the same documents and passages.
- A converter that exits non-zero, or output with no non-blank text (for example a scanned PDF), fails that file with the converter's message and a note that OCR is out of scope.

**Tests:** XLSX and email conversion from small fixtures built in the test (no binary fixture files in the repository), and the missing-converter message (a `PATH` with no `pdftotext`). PDF and DOCX are verified in the walkthrough with the real tools.
