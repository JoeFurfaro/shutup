# shutup — Vision & Open Questions

> ⚠️ This is vague, unvalidated thinking, NOT a plan. It's a snapshot of a
> brainstorm. The core risk (below) is unresolved. Read the risk first. This
> doc is for deciding *whether* to take the demo further — it is NOT build
> guidance. Delete once the repo has enough real context.

## The single biggest open question (read this first)

**Will anyone actually bother switching from `.env`?**

When asked, engineer friends shrugged and said "just use a .env file." None of
them run companies. The honest read: solo devs and tiny teams are mostly *fine*
with `.env` and will accept the AI-context risk rather than adopt a new tool.

The pain likely only gets sharp at specific trigger moments:
- Adding a 3rd teammate / a contractor who needs scoped access
- Someone leaves and everything has to be rotated
- First security questionnaire / compliance ask
- Multiple environments (dev/staging/prod) drifting apart

If that's true, this is NOT a "every dev" product. It's a "small teams past a
certain friction point" product. Smaller market, still possibly real. But the
solo-dev value prop is weak, and a stated goal was to help solo devs too.

**Everything below is only worth pursuing if this question gets a better answer
than "meh, just use .env." Validate with real small teams before building past
the demo.**

## What the product is (one sentence)

A local-first env var + secrets manager that's safe to use alongside AI coding
agents, free for solo/local use, paid when you share with a team.

## Why it might be interesting

- Bounded feature set (appealing to build solo)
- "AI era" relevance without being AI-primary (the AI safety is a thoughtful
  design consideration, not the whole product)
- A real, novel technical primitive: TTY-bypass so agents can use secrets
  without seeing values
- Clear pricing wedge against incumbents (flat pricing vs per-seat)

## The differentiators (vs Doppler, Infisical, 1Password, raw .env)

1. **Agent-safe by design** — TTY-bypass on input, redacted output, agent-aware
   audit, scoped agent tokens. No competitor has this.
2. **Local-first** — works fully offline with no account; cloud is opt-in.
   Competitors are cloud-first. "Secrets aren't on a server to leak from" is the
   strongest version of the safety pitch.
3. **Flat pricing** — e.g. $19/mo for a small team, not $X/seat. Incumbents are
   structurally unable to match this without cannibalizing revenue.
4. **Unified env vars + secrets** — manage all environment config, sensitive or
   not, in one place. Most tools treat secrets as separate.
5. **Personal vs shared vars** — first-class concept. Every team handles this
   badly today (the `.env.example` + "fill in your own" ritual).

## Rough product tiers (very tentative)

- **Free, local-only:** all CLI features, local encryption, agent safety,
  unlimited solo use. A genuine product, not a trial.
- **Pro (~$19/mo):** cloud sync, team sharing (~5 users), web dashboard,
  server-side audit log.
- **Team (~$49/mo):** more users, environments with per-user access control,
  integrations.
- **Business (~$149/mo):** unlimited users, SSO, advanced policies.

Conversion trigger: the moment someone needs to *share* secrets with a teammate.
Solo local use is free forever (free distribution, not lost revenue).

## Strategic stance: control plane, NOT runtime provider

Tempting to be the server that production fetches secrets from at runtime.
Don't — that's the enterprise game (99.99% uptime, multi-region, on critical
path of every deploy, 24/7 on-call, compete head-on with AWS Secrets Manager).
Explicitly NOT wanted.

Instead: be the **control plane** (where humans + agents manage secrets) and
**push to existing providers** (AWS Secrets Manager, Vercel, Railway, Fly, k8s
secrets) that handle runtime. Analogy: be Terraform, not AWS. 1Password, not the
prod auth system.

This keeps operational stakes manageable (if shutup is down, running services
keep working — they pull from AWS, not from you) and avoids the reliability
nightmare that made the on-call/paging idea unattractive.

## How secrets get everywhere (the two patterns)

Don't manually support every injection site. There are only two patterns:

1. **Process env injection:** `shutup run -- <anything>`. Covers local dev,
   local Docker, CI builds (CI runner authenticates with a token). Works for
   every tool that reads env vars at startup.
2. **Push to a cloud secret store:** `shutup sync --target <provider>`. The
   platform's native machinery handles runtime injection. Covers Vercel,
   Railway, Fly, AWS/ECS/Lambda, k8s.

Template rendering (`shutup render <template>`) is the v2 escape hatch for the
weird ~5% that fit neither pattern (custom config files, etc.). Most users
won't need it.

## Scope discipline (lessons from the brainstorm)

The idea got fuzzy when we layered on env vars → templating → cloud sync →
control plane. Each step was logical but the cumulative product lost its wedge.

**The sharp core is:** local-first env/secrets tool with TTY-bypass for agents,
free locally, paid for team sharing. Everything else is *later* and shouldn't
block shipping or even be decided yet. When everything is a feature, nothing is
a wedge.

## Go-to-market (if it ever gets there)

Devs find tools via content + word of mouth, not sales. No cold outreach, no
ads, no booths.
- Build in public while building
- One killer 60s demo video (agent sets up secrets, values never hit the chat)
- Show HN with a broad title (small-team secrets manager, flat pricing) — AI
  safety in the body, not the headline
- One evergreen technical blog post (how agents leak secrets + what to do)
- Open-source a piece (e.g. the TTY/agent-detection bit) for distribution
- Long-term: get mentioned in agent tools' "handling secrets safely" docs

**Positioning caution:** do NOT brand as "the AI secrets manager." Lead with
"great small-team secrets manager that happens to be agent-safe." The AI angle
is the story in specific content, not the brand identity. Avoid gimmick framing.

## Realistic scale

Not venture scale. Ceiling maybe a few thousand small teams at $10–50/mo →
roughly $500K–2M ARR over a few years if executed well. That's a great
side-project / small-business outcome, and that was the stated goal (explicitly
NOT chasing $100M).

## Naming

Working name: **shutup**, domain `shutup.sh` (the `.sh` reads as a shell tool).
"Shut up" = keep quiet = keep secrets; maps to every safety property (don't tell
.env, shell history, the LLM, ex-teammates, logs). CLI reads naturally:
`shutup run -- npm start`. Slightly cheeky; fine for the small-team dev audience,
maybe less so for enterprise procurement (a later-you problem).

## Agent integration approach

- v1: append a marked instruction block (between `<!-- shutup:start/end -->`
  delimiters so it's updatable) to `CLAUDE.md`, `.cursorrules`, `AGENTS.md`.
  Cover Claude Code, Cursor, Codex + a generic fallback (~95% of the market).
- Self-healing: when the CLI detects an agent context and the agent does
  something wrong (reads .env, tries to print a secret), the error message
  teaches the correct pattern.
- v2: ship an MCP server so agents get native `shutup.run` / `shutup.set` /
  `shutup.check` tools with safety constraints built in. Likely becomes the
  primary integration as MCP support matures; markdown becomes the fallback.

## Other open questions / risks

- Doppler could ship "agent-mode" in a quarter and erase the sharpest
  differentiation. Defense: move faster, stay more focused, be cheaper. Not a
  moat, a head start.
- Is the agent-leak threat sharp enough *today*, or are we ~6–12 months early?
  Early is good if the wave is coming (it probably is), bad if it isn't.
- TTY-bypass works great for terminal agents (Claude Code, Cursor, Codex,
  Aider). Sandboxed or pure-API agents without real TTY access fall back to
  stdin (less safe) — need to detect and warn.
- Output redaction (scrubbing secret values from a child process's stdout when
  run under an agent) is best-effort and harder than input protection. The
  input/prompt case is the clean, bulletproof one; lead with that.
- Marketing/content is the real bottleneck, not the engineering. Most engineers
  underestimate this. Budget real time for it.

## Architecture seeds (so the demo doesn't paint us into a corner)

> The CLI demo is built (see CONTEXT.md for the as-built spec). Notes below mark
> what's realized vs. still future.

- **[built]** `Store` interface in the CLI from day one. Demo uses `LocalStore`
  (plaintext JSON at `~/.shutup/<id>/store.json`); cloud version later swaps in
  `APIStore` — or an encrypting wrapper — behind the same interface. The command
  layer never touches storage details directly.
- **[built]** Secure by default + config/store split: the committed
  `shutup.config.yaml` holds the contract + public values; secret values live in
  the per-machine, per-project store keyed by `proj_<id>`, outside any repo. That
  `id` is also the future API's primary key for the project.
- **[future]** Local file becomes an *encrypted* cache, NOT the source of truth.
  v1 is plaintext (the home-dir location already removes the "secrets committed to
  the repo" risk, and the TTY-bypass — not at-rest encryption — is the actual
  safety mechanism). Encryption (AES-GCM, key in OS keychain) slots in behind the
  `Store` interface; there's a `TODO: encrypt` marker there. In the cloud version
  the source of truth is the DB/API; the cache makes `run` fast + offline-capable.
- **[open question] Shared scope across projects in a repo.** v1 models
  `project → env → vars`: each project (one `shutup.config.yaml` + `proj_id`) owns
  its own store, and envs are nested *inside* a project. In a monorepo where
  `frontend` and `backend` share secrets (same `DATABASE_URL`, same
  `STRIPE_SECRET_KEY`) but each needs only a subset, this duplicates values and lets
  them drift. The likely fix is to **decouple env from project**: treat an env as a
  named value-scope that multiple projects can draw from, with each project declaring
  the *subset* it requires (and optional project-local overrides). Sharing should be
  **opt-in per var** — this is the same primitive as "personal vs shared vars," just
  with projects (not people) as the subjects. Deferred: it doesn't corner us
  (resolution lives in one place, `project.Resolve`, and a config already references
  an id, so a shared scope is just another id to point at), and it overlaps the
  cloud/team work. Validate the monorepo-sharing pain before building it.
- Schema-driven type safety across components (later, when there's a backend):
  Django Ninja → OpenAPI → generated TS client (dashboard) + Go client (CLI).
  Commit generated files; CI fails on drift. Overkill now, great once there are
  three codebases sharing an API.

## Stack (for if/when it goes past the CLI demo)

- CLI: Go (single static binary, great distribution via GoReleaser + Homebrew
  + curl install)
- Backend: Django + Django Ninja (familiar, fast to ship, free admin for
  support/debugging — lock it down, internal only)
- Dashboard: Vite + React + Tailwind + shadcn/ui (NOT Next.js — no SSR needed
  for an authed SPA)
- DB: Postgres
- Auth: hosted (Clerk / Supabase Auth) — don't build it
- Monorepo, plain Makefile (skip Nx/Turborepo/Bazel — unnecessary at this size)

## Bottom line

The demo is worth building for fun and to feel out the core workflow. Whether it
becomes a business hinges on the question at the top of this doc. Build the demo,
use it on a real project, show it to small teams that have actually hit the
trigger moments — then decide. Don't build past the demo on conviction alone.