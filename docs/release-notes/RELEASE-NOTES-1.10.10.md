# Spekk CLI 1.10.10 — Fix Broken Markdown in `spekk show --watch`

`spekk show --watch` could render as a wall of raw JavaScript instead of the spec explorer — the bundled markdown library leaked onto the page as visible text and no markdown rendered.

## Cause

Watch mode injects a live-reload `<script>` before the page's closing `</body>`, using a first-match string replace. That was safe for two years — until a vendored update to the bundled DOMPurify sanitizer added a build whose minified source contains a literal `"</body></html>"` string (its XHTML-parsing branch), *before* the real closing tag. The first-match insert then dropped the reload `<script>` inside DOMPurify's own `<script>` block, and its closing `</script>` prematurely terminated that block — spilling the rest of the library into the page as text. Nothing in watch mode changed; a library bundle silently tripped a latent assumption that "the first `</body>` is the document's."

## Fix

The reload client now anchors on the **last** `</body>` in the document, which is always the real one. Plain `spekk show` (no `--watch`) was never affected — it injects nothing.

## Upgrade

```bash
spekk update
```
