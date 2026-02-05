---
id: contributing-documents-publish-process
parent: gemfury-package-management
created: 2026-02-05T12:00:00Z
priority: 1
status: not_started
---

# CONTRIBUTING.md Documents Publish Process

## What Must Be True

A CONTRIBUTING.md file exists at the repository root that documents how maintainers publish new versions of @spekk/cli to GemFury.

## Success Criteria

- [ ] CONTRIBUTING.md file exists at repository root
- [ ] Documents how to set up a GemFury publish token (different from read token)
- [ ] Documents versioning conventions (semver)
- [ ] Documents the publish command/process
- [ ] Documents any pre-publish checklist (tests pass, changelog updated, etc.)

## Notes

The publish process likely involves:
1. Bumping version in package.json
2. Running tests
3. Publishing to GemFury via npm publish or fury CLI
4. Creating a git tag

The builder should investigate the actual GemFury publish workflow and document it accurately.
