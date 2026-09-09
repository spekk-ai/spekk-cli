---
id: destroy-keeps-operator-keys
parent: sandbox-provider-abstraction
created: 2026-09-05T14:00:00Z
priority: 1
status: done
depends-on: register-an-existing-machine
branch: fix/destroy-keeps-operator-keys
---

# Destroy Deletes a Key Pair Only When It Is a File in Spekk's Own Keys Directory

`spekk sandbox destroy` removes the local SSH key pair of the sandbox it tears down. That is correct for a key spekk generated: nothing else uses it, and the machine it opened is gone. It is not correct for a key the operator supplied. A droplet created with `--ssh-key ~/.ssh/id_rsa`, or a machine registered with `--provider none --ssh-key ~/.ssh/work`, records the operator's own key, and a destroy that deletes it is not recoverable.

The rule that decides has to be stated in terms of the file, not the record. Spekk writes every key it generates into the `keys` directory under its config dir, so "spekk generated this key" and "this key is a file in that directory" are the same claim, and the second one can be checked.

## Success Criteria

- Destroy deletes the private key and its `.pub` only when the recorded path, cleaned and made absolute, is inside spekk's `keys` directory, and the file it resolves to, with symlinks followed on both sides, is also inside that directory once its own symlinks are resolved. Any other path is kept.
- A symlink is kept in both directions: a link inside the keys directory that points out of it, and a link outside the directory that points into it. Both are the operator's arrangement, not spekk's.
- A sibling directory that shares the prefix of the keys directory does not pass. The separator is part of the comparison.
- A recorded path with nothing on disk at it, inside the keys directory, counts as spekk's. There is nothing at it to protect, and the removal is a no-op.
- When destroy keeps a key, it prints one line to stderr that names the path, so the operator knows the file is still there. A key it deletes is not reported as kept.
- A record for a machine no cloud owns (`provider: none`) has no droplet. Destroy on it does not build a DigitalOcean client, needs no `DO_API_TOKEN`, and sends no delete for droplet 0. The command layer resolves the provider from the record, and a nil provider is what keeps destroy away from the API.
- Destroy of a record that names a droplet is unchanged: it deletes the droplet and the DigitalOcean key, then the local files, then the record.

**Tests:** `internal/sandbox/provider_test.go` (`TestOwnsKeyPair`, `TestDestroyRemovesOnlyGeneratedKeys`), `internal/sandbox/unmanaged_test.go` (`TestDestroyUnmanagedMachine/needs_no_cloud_token_and_calls_no_cloud`)
