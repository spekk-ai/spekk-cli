# Spekk CLI 1.30.0 - A Sandbox on AWS, and a Deploy That Replaces a Running Agent

A spekk sandbox needed a DigitalOcean droplet or a machine you prepared yourself. This release adds a CloudFormation template that stands one up on AWS in a single stack. It also fixes the deploy path, which could not replace an agent that was running, and makes the observer agree with itself about which findings are still open.

## A sandbox on AWS

One `aws cloudformation create-stack` creates an isolated VPC, a public subnet, a security group, and one Ubuntu 24.04 instance prepared to the state a droplet reaches. The stack outputs the public address and the `spekk sandbox create` command that registers it, ready to paste. One `delete-stack` removes everything the stack made.

Spekk does not provision a machine it did not create, so the instance has to arrive prepared. The template's UserData is therefore the same cloud-init content a droplet gets, with two differences: the agent user's key comes from a parameter, and the users list opens with `default`, which keeps the image's `ubuntu` login that the key pair is for. The copy is not maintained by hand. `go generate ./internal/sandbox` writes the block from `cloud-init.yaml`, and a test fails when the two files differ in any other way, when `cloud-init.yaml` gains a `${` that `Fn::Sub` would read as a variable, or when it has more than one users list.

The SSH source is a required parameter with no default, so nobody opens port 22 to the internet by accident. The register command the stack prints carries `--ssh-user ubuntu`, because an AWS Ubuntu image refuses a root login. The new page [A Sandbox on AWS](../advanced/sandbox-aws.md) walks the whole path.

## A deploy replaces an agent that is running

`spekk sandbox deploy` copied the new binary over the installed one. A file that is executing cannot be written, so a deploy to a live sandbox failed, and the documented workaround was to stop the service and move the file by hand.

Every login now stages the upload in its home directory, then prepares the executable on the destination filesystem and renames it into place. A rename is atomic, and a running process keeps its old executable until the service restart, so the replacement is safe while the agent is working. A failure during the upload or the preparation leaves the installed binary alone and never reaches the service commands. This closes the deploy half of issue #214, and the manual workaround is gone from the release page.

## One rule for which findings are still open

Announce, dedup, and the digest each decided whether a finding was still live, and they did not agree. Announce matched observer branches with a SQL pattern in which `%` also matches a slash, so an observation whose slug was `main` could be treated as resolved by one surface and open by the other two.

There is one rule now, in one function, and all three call it. An observation is a live claim when it sits on the `observer/<slug>` branch named after it and no observation with that slug has reached `main`. The three surfaces cannot drift apart, because there is nothing left to drift.

`spekk observer scan-check` also refuses a `--type` outside the two valid values, and refuses a stray argument. Both were silent before: the command answered `clear`, the agent filed the observation, and the parser dropped it on read with nothing reported. A path list written with spaces instead of commas was the common way to lose one.

## `spekk list --priority`

`spekk list --priority 1` keeps the assertions at that priority. It combines with `--status`, works in the table, JSON, TSV, and CSV output, and accepts any nonnegative integer. A value outside the stored range returns an empty result in the format you asked for rather than an error.

## Arguments fail where they used to pass

Three argument mistakes used to run anyway, each discarding part of what you typed.

A value written as `--flag=value` was an unknown token, so `spekk list --priority=1` returned the unfiltered list and exited 0. It now fails. A flag that takes a value and is supplied twice kept the last one, so `--status draft --status done` silently dropped the first. It now fails, on every command, because the parse result holds one value per flag and a repeat can only lose one. A flag that takes no value may still repeat, because a repeat discards nothing there.

The errors also print where a caller can read them. A bad filter value prints as JSON on stdout, so a script that asked for `--json` reads the error in the format it expects. A malformed command line prints as text on stderr, as the mutually exclusive format flags always did.
