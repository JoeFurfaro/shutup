# shutup — CLI Context (as built, v2 model)

> ℹ️ This describes the `shutup` CLI **as built** in `cli/`. It supersedes the original
> spec and the earlier v1 model. The design was refined over a long pass; the v2 model
> below (envs own values; projects consume names) is what the code implements. For the
> bigger-picture product vision, see VISION.md. If the two conflict, this doc wins *for
> the demo*.
>
> **If you are an agent spinning up a demo app that USES shutup, jump to
> "Using shutup in a project".**

## What this is

`shutup` is a local-first env var + secrets manager. This demo is **local-only** — no
backend, auth, or cloud. The goal is to prove the core workflow feels good, especially
the **"an AI agent can use secrets without seeing their values"** mechanic. We built
**only the CLI**, in Go. A single static binary; envs live on the machine.

## The mental model (read this first)

- An **environment** ("env", e.g. `dev`/`prod`) is a bag of variables — each with a
  **value** and a **secret/public flag**. Envs hold *all* the values and the visibility,
  and live on your machine (`~/.shutup/envs/<id>.yaml`), **not** in the repo. An env is
  an anonymous, id-keyed bag.
- A **project** (the dir you're in, found by walking up to `shutup.config.yaml`) just
  declares the variable **names** it consumes, maps local env names → env ids, and picks
  a default env. It holds **no values, no secrets** — safe to commit.
- **Identity = env id; names are local labels.** `dev` is a per-project nickname for an
  env id. Two projects (same repo or different repos) **share** an env by pointing at the
  same id. This dissolves "monorepo vs multi-repo" — there's just projects referencing
  envs by id.
- **git holds declarations; envs hold values.** A fresh clone gets the manifest, not the
  values; `missing` drives setup. Cross-person *value* sharing is deliberately the API's
  job (the free/paid line) — but a **secret-free `env export`/`env import` bundle** lets
  you hand an env's structure + public values to a teammate out-of-band today.
- **Context-based, like git.** You work from inside a project; `--env` picks the env
  (default: the project's `default_env`). No project names/flags/paths.

## The key differentiator (the "wow")

**TTY bypass.** `shutup set STRIPE_KEY` reads the value directly from `/dev/tty` with
echo off — NOT from stdin. So when an AI agent invokes the command, the value the user
types goes keyboard→CLI; it never enters the agent's stdout/stderr/context. shutup
**never falls back to stdin** for secret input — no terminal → it refuses.

After `shutup set`, ask the agent "what value did I enter?" — it genuinely can't know.
(Honest scope: this protects *input*. The store is plaintext on disk in v1 and the agent
runs as you, so it's a strong guardrail against incidental leakage, not a sandbox against
a hostile agent. See VISION.)

## Data model

**Env** — `~/.shutup/envs/<env-id>.yaml` (local; API later via `source`). Local ids are
`envlocal_<32hex>`; `env_` is reserved for future API-backed envs.
```yaml
id: envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47
source: local
vars:
  DATABASE_URL: { value: "...", secret: true }
  PORT:         { value: "3000", secret: false }
```

**Project** — committed `shutup.config.yaml` (declarations only):
```yaml
consumes: [DATABASE_URL, PORT]   # var NAMES this project needs (visibility-agnostic)
envs:
  dev: envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47   # local name -> env id
default_env: dev
```

Validity: `default_env` must be a key in `envs`; `envs` values must be valid env ids. A
consumed var with no value in the resolved env is *incomplete* (surfaced by `missing`),
not invalid.

## Commands

Resolve the current project by walking up from cwd; `--env <name>` resolves via the
project's `envs` map, defaulting to `default_env`.

- `shutup init [--link <env-id>]` — create `shutup.config.yaml` in cwd with a default env
  named `dev` (fresh, or `--link` an existing id to share). Inject the CLAUDE.md block.
  Hard-blocks if an ancestor project exists.
- `shutup set <NAME> [value] [--public] [--env <n>]` — write NAME's value to the env
  (secret via hidden prompt; inline only with `--public`) and auto-wire it into
  `consumes`. Never prints the value.
- `shutup use <NAME>` / `shutup unuse <NAME>` — add/remove a name from `consumes` (no
  value). `use` is for consuming an already-set shared var.
- `shutup missing [--env <n>]` — consumed vars with no value in the env (names to stdout).
- `shutup list [--env <n>]` — consumed vars + state; public values shown, secrets only as
  `(secret, set)`.
- `shutup check <NAME> [--env <n>]` — `exists`/`not found` (exit 0/1), never the value.
- `shutup run [--env <n>] -- <cmd…>` — inject **only** consumed vars (least-privilege) and
  exec; stdio/signals/exit-code forwarded.
- `shutup env add <name> [--link <id>]` · `env ls [--all]` · `env default <name>` ·
  `env rm <name> [--delete]` — manage which envs a project uses.
- `shutup env export [<name>] [-o <file>]` — write a **secret-free** bundle (id + public
  values + secret *names*, never secret values) to share with a teammate.
- `shutup env import <file> [--as <name>]` — merge a bundle into your store by id (keeps
  your local secrets); `--as` also links it into the current project.
- `shutup import <file> [--public a,b] [-i] [--delete] [--env <n>]` — migrate a `.env`.
  **Bare = discovery (lists names only, never reads values into context)**; acting needs
  `--public` or `-i`. Wires imported names into `consumes`.
- `shutup destroy [--yes]` — remove the project (config + CLAUDE.md block). Does **not**
  delete shared envs (use `env rm --delete` for those).

## Using shutup in a project (for the demo-repo agent)

1. **`shutup init`** at the project root. Commit `shutup.config.yaml`.
2. **Add what the app needs** with `shutup set`:
   - Secrets (default, hidden prompt): `shutup set DATABASE_URL`, `shutup set STRIPE_SECRET_KEY`.
   - Public (inline): `shutup set PORT 3000 --public`, `shutup set NODE_ENV development --public`.
     (Anything `NEXT_PUBLIC_*` ships to the browser → genuinely public → `--public`.)
   - Or migrate an existing file: `shutup import .env` (lists names) → `shutup import .env --public PORT,NODE_ENV --delete`.
3. **Wrap scripts with `shutup run --`:**
   ```jsonc
   "scripts": {
     "dev":     "shutup run -- next dev",
     "build":   "shutup run -- next build",
     "start":   "shutup run -- next start",
     "migrate": "shutup run -- node scripts/migrate.js"
   }
   ```
4. **Docker:** public config as `--build-arg` (e.g. `NEXT_PUBLIC_*`); secrets injected at
   **run** time, never baked into the image:
   ```sh
   shutup run -- docker build --build-arg NEXT_PUBLIC_APP_URL -t myapp .
   shutup run -- docker run --rm -p 3000:3000 \
     -e DATABASE_URL -e STRIPE_SECRET_KEY -e PORT -e NODE_ENV myapp
   ```
5. **Monorepo (multiple apps):** each app dir is its own project (`shutup init` in each).
   Share an env by linking the same id: `shutup env add dev --link <id>` (find ids with
   `shutup env ls --all`), then `shutup use <NAME>` for the shared vars. Each project's
   `run` injects only what it consumes.
6. **Onboarding a teammate:** they clone (config maps `dev` → the shared env id), you send
   them `shutup env export dev -o dev.bundle`; they `shutup env import dev.bundle`, then
   `shutup missing` shows the secrets to set (their own values — secrets never leave a
   machine in v1).
7. **Keep the CLAUDE.md block** `init` writes — it teaches agents to use this interface.

## Architecture

The **`EnvStore` interface** (`internal/env`) is the swappable seam — `LocalEnvStore`
today, a cloud `APIStore` or encrypting wrapper later, no command changes.
```
cli/internal/
  cmd/      cobra commands (agent-legible help is a first-class deliverable)
  config/   shutup.config.yaml: consumes + envs(name→id) + default_env
  env/      Env/Var + EnvStore + LocalEnvStore (~/.shutup/envs/<id>.yaml) + bundle export/import
  project/  ties config + env store: resolve, missing, set (auto-wire), run (only consumed)
  dotenv/   .env parser for `import`
  tty/      /dev/tty hidden + visible prompts (the wow); Unix in v1, CONIN$ seam for Windows
  agent/    the CLAUDE.md instruction block
  id/       envlocal_/env_ id generation + validation
  ui/       TTY-aware colored output
```
No at-rest encryption in v1 (`TODO: encrypt` seam behind `EnvStore`). cgo-free for clean
cross-compilation / distribution later.

## Out of scope (v1)

Cloud/API, auth, live cross-person secret *value* sync, access control, rotation, audit,
Windows (`CONIN$` seam left), packaging — all deferred. The local + agent-safe + team-
onboarding layer is the product; sharing live secret values is the API (the paywall).
