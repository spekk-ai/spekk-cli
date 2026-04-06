---
id: go-sandbox-api-client
parent: golang-sandbox
created: 2026-04-05T12:28:00Z
priority: 2
status: done
depends-on: go-command-router
branch: feature/golang-sandbox
---

# Go DigitalOcean API client

A Go HTTP client for the DigitalOcean API that handles droplet CRUD, SSH key management, project assignment, and VPC placement.

## Success Criteria

- Creates droplets with cloud-init user data
- Lists droplets filtered by spekk tag
- Gets droplet status by ID
- Destroys droplets by ID
- Manages SSH keys (create, list, find by fingerprint)
- Assigns droplets to DigitalOcean projects
- Places droplets in specific VPCs
- Uses `DIGITALOCEAN_TOKEN` environment variable for auth
- Proper error handling for API failures (rate limits, not found, auth errors)
