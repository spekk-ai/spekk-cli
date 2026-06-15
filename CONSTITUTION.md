# The Spekk Constitution

This document states the values that govern Spekk CLI. It is not a style guide and not a roadmap. It is the set of commitments we measure decisions against — what we build, what we refuse to build, and how we resolve the cases where those commitments pull in different directions.

When a proposal — a feature, a dependency, a redesign — is on the table, the question is not "is this useful?" Most things are useful. The question is "does this hold up against the values below?"

---

## Principles

### 1. Low to no dependencies

Every dependency is a liability we inherit: its bugs, its breaking changes, its supply-chain risk, its maintenance burden. We add one only when the cost of writing and owning the equivalent code ourselves is clearly higher than the cost of the dependency over its whole lifetime.

**In practice**
- The standard library is the default. Reach outside it only with a deliberate reason.
- Prefer talking to external systems over plain protocols (HTTP, JSON, stdio) rather than via vendor SDKs. An SDK per provider is the fastest way to grow the dependency tree (see Principle 6).
- Adding a dependency is a reviewable decision, not a convenience. Removing one is always welcome.
- We do not carry two libraries for the same job.

**Non-goals**
- We are not chasing zero dependencies as a trophy. A well-chosen dependency beats a poorly-maintained reimplementation.

### 2. A single, self-contained binary

Spekk is one binary you can drop onto a machine and run. No runtime to install, no package manager to satisfy, no companion services required for the core workflow.

**In practice**
- Assets the tool needs are embedded, not fetched at runtime.
- The core workflow (`init`, `next`, `coach`, `builder`) works from the binary alone.
- Distribution stays simple: download, `go install`, or a one-line installer.
- It should start instantly and stay quiet — no surprising background work.

**Non-goals**
- We are not building a daemon, a service mesh, or a platform you have to operate.

### 3. Free and open source, developed in the open

Spekk is Apache-2.0 licensed and stays that way. "Open" means more than the license file — it means the development process itself is visible and the tool's behavior is honest.

**In practice**
- No hidden behavior: no telemetry, no analytics, no network calls the user did not ask for. If the tool talks to the network, it is because the user invoked something that obviously requires it.
- Decisions, specs, and history are public.
- The license stays permissive.

**Non-goals**
- No closed "pro" core. No data collection as a business model.

### 4. Spec-driven, applied to ourselves

Spekk is a tool for spec-driven development, and we build it that way. We use the tool to build the tool. This is the strongest test we have: if the workflow is painful on our own repo, it is painful for everyone.

**In practice**
- Behavior is described in `specs/` before or alongside the code that satisfies it.
- We feel our own rough edges first, and fix them.
- Dogfooding is a requirement, not a demo.

**Non-goals**
- This is a working practice, not a purity test. We are not above a quick fix when the situation calls for one — but the spec follows close behind.

### 5. Minimalist — one thing, done well

Spekk turns a `specs/` directory into a work queue for AI agents. That is the thing. We would rather do that one thing excellently than do ten things adequately. Every feature is weight: to learn, to document, to maintain, to not break.

**In practice**
- The default answer to "can it also do X?" is no, until X is clearly part of the one thing.
- Integrations with other tools should be thin and generated, not bespoke logic we maintain by hand. Breadth of reach must not become depth of burden.
- We say no to good ideas that belong in a different tool.
- Platform scope is deliberate: macOS and Linux are first-class; Windows is best-effort. This is a minimalism decision, stated plainly rather than apologized for.

**Non-goals**
- We are not building a Swiss Army knife. "A little bit of everything for everyone" is the failure mode we are guarding against.

### 6. Open mindset — model and provider agnostic

Spekk does not marry a model or a provider. The AI landscape shifts monthly; a tool welded to one vendor is obsolete by design. The user's choice of model and assistant is theirs, not ours.

**In practice**
- Talk to providers over open, stable interfaces — not lock-in SDKs (this is also how we honor Principle 1).
- Works inside whichever assistant the user already has, and standalone from the terminal.
- No provider gets special treatment baked into the core.

**Non-goals**
- We are not a frontend for one company's models. We are not picking winners.

### 7. A high bar for quality

A feature works and works well, or it does not ship. A feature that has quietly stopped working well is a candidate for removal, not a permanent resident. Half-working features are worse than absent ones: they erode trust in everything else.

**In practice**
- Shipped behavior is backed by specs that pass. A feature without passing specs is, by definition, a candidate for removal.
- The spec format is a contract other repositories depend on. We treat its stability with the seriousness that implies, and we are explicit about what is stable versus still in motion.
- "Remove it until it works" is a real, available decision — not a threat we never carry out.

**Non-goals**
- Quality is not gold-plating. The bar is "works and works well," not "polished beyond what the one thing needs."

---

## Holding the values in balance

These principles will conflict. Low dependencies pulls against provider-agnosticism. Minimalism pulls against reach. Quality pulls against shipping. That tension is the point — it is what keeps any single value from running away with the project.

There is no fixed ranking. We do not have a rule that says "Principle 1 always beats Principle 6." Conflicts are resolved case by case, in the open, with the trade-off named out loud. A decision that sacrifices one value for another should say so explicitly, so the cost is visible and can be revisited.

The discipline this asks of us is simple: when you make a call, name which values you are trading against each other and why. "We added this dependency because owning the equivalent would cost more over its lifetime" is a constitutional argument. "It was easier" is not.

---

## Amending this document

This constitution changes the way the principles themselves work: slowly, deliberately, and in the open. Proposing a change means making the case in public and showing how the new wording holds up against the values already here. If a principle no longer reflects how we actually build, we change the principle or change the behavior — we do not let the two quietly drift apart.
