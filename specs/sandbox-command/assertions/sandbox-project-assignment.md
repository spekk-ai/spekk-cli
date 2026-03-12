---
id: sandbox-project-assignment
parent: sandbox-command
created: 2026-03-12T21:00:00Z
priority: 2
status: not_started
depends-on: sandbox-create-workflow
branch: feature/sandbox-command
---

# Sandbox Project Assignment

## Requirement

`spekk sandbox create` accepts an optional `--project` flag to assign the new droplet to a DigitalOcean project. Since the DO API doesn't support project assignment at droplet creation time, the droplet is moved into the project immediately after creation.

## Success Criteria

- `src/sandbox/do-api.js` exports a `listProjects()` function that GETs `/v2/projects` and returns all projects
- `src/sandbox/do-api.js` exports an `assignToProject(projectId, resourceUrns)` function that POSTs to `/v2/projects/{projectId}/resources` with the resource URNs
- `spekk sandbox create --name foo --project "My Project"` looks up the project by name from `listProjects()`, creates the droplet, then assigns it to that project using URN `do:droplet:{dropletId}`
- If `--project` value matches a project UUID instead of a name, it uses the ID directly without looking up by name
- If `--project` is specified but no matching project is found, prints an error listing available project names and exits with code 1 (before creating the droplet)
- If `--project` is not specified, no project assignment happens (droplet goes to the default project, same as current behavior)
- The `--project` flag is documented in `spekk sandbox create --help` output
- Project name (if specified) is saved in the local sandbox metadata store alongside existing fields
