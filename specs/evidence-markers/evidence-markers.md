---
id: evidence-markers
created: 2026-11-02T21:30:00Z
priority: 1
---

# Evidence Markers - Each Check Names the Property It Checks

## Problem

Verification gives an agent permission to skip the code behind a property. For this, spekk must know which tests check which property. The link goes from the check to the property, so that the link moves with the test: a renamed or moved test keeps its marker, and a deleted test takes its marker with it.

This spec covers the markers for Go and for Python (pytest), and how spekk finds them. `verify-command` maps test results to properties through them.

## Marker syntax

Go has no decorators. A Go marker is a `spekk:` comment in the doc comment directly above a test function:

```go
// spekk: balance-nonneg evidence=property
func TestBalanceNeverNegative(t *testing.T) { ... }
```

Python uses a decorator on a pytest test function:

```python
@spekk("balance-nonneg", evidence="property")
def test_balance_never_negative(): ...
```

A pytest project can put this no-op definition in `conftest.py` so the decorator leaves each test callable:

```python
def spekk(property_id, *, evidence="example"):
    def mark(test):
        return test
    return mark
```

The scanner reads the source text. It does not import `conftest.py` or run a test to find a marker. A test file can use `from conftest import spekk` to make the decorator available to pytest.

## Decisions

- The evidence link goes from the check to the property (William, 2026-11-01).
- The Python marker is `@spekk("<id>", evidence="<level>")`.
- Go uses a comment marker above the test function. spekk-cli's own tests are in Go, so this is necessary to dogfood v2.
- The evidence levels, from weak to strong, are `example`, `property`, `contract`, `bounded` and `proof`. Spekk calculates the level of a property from its passing checks (`verify-command`). It does not trust a declared level.
- TypeScript and Rust markers are later work.

## Open choices

- **Static scan.** Spekk finds markers by a read of the source text. It does not import or run the code to find them.
- **The Python `spekk` name.** v2.0 ships no Python package. The project defines `spekk` as a decorator that does nothing, for example in `conftest.py`. The spekk docs give the short definition. A package can come later.
- **No `evidence`.** A marker with no `evidence` has level `example`.
- **Go functions.** A Go marker can mark a `Test` or a `Fuzz` function. A fuzz function with its seed corpus is a common Go property check.
- **The Go marker form.** A `spekk:` comment directly above a test function is a marker, also with no `evidence=`. A `spekk:` comment at any other place is an anchor (`comment-anchors`).

## Assertions

See `assertions/` for what must be true.
