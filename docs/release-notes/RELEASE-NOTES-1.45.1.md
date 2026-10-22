# Spekk CLI 1.45.1 - Safer Sandbox Attachment Downloads

The sandbox agent now checks each attachment before it starts a model turn. It downloads files from the authenticated control host route into a private directory for that turn and gives the saved paths to the model. A failed download stops the turn, reports an error, and removes files from that incomplete batch.

The agent accepts at most 10 attachments, 20 MiB for one file, and 40 MiB for one turn. It requires a valid attachment ID, a safe filename, an HTTP 200 response, and a nonempty file. It rejects redirects and files that exceed the limits. The download uses a 30 second HTTP timeout and stops when the turn is canceled. The control host can set lower upload limits.

This release does not enable file input on the control host. File input needs separate approval. Real model file and image reads remain an immediate post-deployment QA check. The revision also corrects the global skill directory paths in the specs; it does not change skill commands or their defaults.

## Upgrading

After the release is published, use `spekk sandbox deploy <name>` to install the new agent on each authorized sandbox. Confirm the installed CLI and agent versions with `spekk sandbox status <name>`.
