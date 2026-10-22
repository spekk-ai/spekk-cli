# Spekk CLI 1.45.2 - Clean Repository Checkouts for Attachments

The sandbox agent saves downloaded attachments in its configured workspace, outside a prepared repository checkout. It gives the file paths to the model for the current turn. A later turn can select a ref and prepare the same repository checkout because the attachments do not change it.

User edits in the checkout still cause repository preparation to refuse the checkout. The attachment download limits and safety checks from 1.45.1 remain in effect.

File input on the control host remains off pending separate approval. Real-model file and image reads remain open as immediate post-deployment QA. This release does not claim that acceptance check has passed.

## Upgrading

After the release is published, use `spekk sandbox deploy <name>` on each authorized sandbox to install the new agent. Confirm the installed CLI and agent versions with `spekk sandbox status <name>`.
