# Spekk CLI v1.2.4 — Live Reload + Searchbar

Two quality-of-life improvements for the spec explorer:

## `spekk show --watch` (new)

Run `spekk show -w` and the spec explorer now live-reloads in your browser as specs change. No more re-running `spekk show` after every edit. Uses SSE for instant updates, preserves your UI state (expanded specs, selected panel, scroll position) across reloads. Ctrl+C to stop.

## Searchbar filtering

The spec tree now has a search input that filters specs and assertions in real-time by name, status, or priority. Matching completed specs are surfaced even when hidden by the toggle.
