---
slug: conversation-files-spec-omits-total-size-limit
type: outdated_specs
severity: medium
status: open
created: 2026-11-21T12:00:00Z
announced: 2026-11-21T20:02:34Z
affected:
  - specs/agent-file-delivery/assertions/conversation-files-command.md
  - specs/agent-file-delivery/assertions/agent-current-thread-reply.md
  - cmd/spekk/conversation.go
  - docs/cli-reference.md
---

# Conversation file specs omit the 20 MB combined size limit

## Issue Description

The `conversation-files-command` assertion lists the CLI's checks before it sends anything: a file over 10 MB, more than three files, and an absent session. The CLI makes one more check. It refuses when all attached files together are over 20 MB. `spekk conversation open` and `spekk conversation reply` both run this check, and the usage text and the CLI reference both document it. Neither assertion states it.

## Evidence

- `specs/agent-file-delivery/assertions/conversation-files-command.md`: "The CLI refuses a file over 10 MB, more than three files, and an absent session before sending." There is no combined limit.
- `specs/agent-file-delivery/assertions/agent-current-thread-reply.md`: "zero to three files". There is no size limit at all.
- `cmd/spekk/conversation.go`, `deliverConversation` (around line 283): `if total > 20<<20 { ... "Error: all files together must be at most 20 MB" }`. Both `execConversationOpen` and `execConversationReply` use this path.
- `cmd/spekk/conversation.go`, the `conversationOpenUsage` and `conversationReplyUsage` help text: "all files together must be at most 20 MB."
- `docs/cli-reference.md` (around line 1168): "Each file is at most 10 MiB, and all files together are at most 20 MiB."
- `cmd/spekk/conversation_test.go` tests the 10 MB single-file case (`Truncate(10<<20 + 1)`) but not the combined limit.

## Impact

Users see this limit in the help text and the docs, and it changes when the command fails. Three 9 MB files pass every check the spec states, but the CLI refuses them. Only the code and the docs hold the rule, and no test covers it. A builder that reworks delivery to match the assertions could remove the rule and still meet every stated criterion.

## Recommendation

Add the combined limit to both assertions. They should say that the CLI refuses attachments over 20 MB in total before it sends anything, and give the exact error. The remedy commit on this branch makes that edit. If the limit was not intended, delete this branch and file the code side instead.