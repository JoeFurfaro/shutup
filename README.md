# shutup

A local-first environment variable + secrets manager that's **safe to use alongside
AI coding agents** — an agent can *use* your secrets without ever *seeing* their
values.

> Local-only today (no backend/cloud). The CLI lives in [`cli/`](cli/). For the
> product direction and open questions, see [VISION.md](VISION.md).

## The "wow": TTY bypass

`shutup set STRIPE_KEY` reads the value straight from the terminal (`/dev/tty`) with
echo off — **not** from stdin. So when an AI agent runs the command, the value you
type goes keyboard → CLI; it never enters the agent's stdout/stderr/context. shutup
never falls back to stdin for secret input — no terminal, it refuses.

After `shutup set`, ask the agent "what value did I enter?" — it genuinely can't know.

## Install

```sh
cd cli
make install          # builds and installs to ~/.local/bin/shutup
# or pick a location:
make install SHUTUP_BIN=/usr/local/bin/shutup
# or by hand:
go build -o ~/.local/bin/shutup .
```

Requires Go 1.24+. The binary is a single static, cgo-free executable.

## Quickstart

```sh
cd my-app
shutup init                         # creates shutup.config.yaml + a default "dev" env
shutup set DATABASE_URL             # hidden prompt — value never shown or logged
shutup set PORT 3000 --public       # non-sensitive value, inline
shutup list                         # PORT shown; secrets shown only as (secret, set)
shutup missing                      # what's declared but not set yet
shutup run -- npm start             # injects this project's vars into the process
```

Already have a `.env`? Migrate it:

```sh
shutup import .env                              # lists the var NAMES (never the values)
shutup import .env --public PORT,NODE_ENV --delete   # classify by name, import, remove .env
```

## Mental model

- An **environment** (`dev`/`prod`) is a bag of variables — each with a **value** and a
  **secret/public flag**. Envs hold all the values and live on your machine
  (`~/.shutup/envs/<id>.yaml`), **not** in the repo. An env is an anonymous, id-keyed bag.
- A **project** (the dir you're in, found by walking up to `shutup.config.yaml`) just
  declares the variable **names** it consumes and which envs it uses. No values, no
  secrets — safe to commit.
- **Identity = env id; names are local labels.** Two projects (any repos) share an env by
  pointing at the same id. There's no "repo" concept — just projects referencing envs.
- **git holds declarations; envs hold values.** A fresh clone gets the manifest, not the
  values; `shutup missing` drives setup. (Live cross-person value sync is the future API;
  a secret-free `env export`/`import` bundle covers ad-hoc hand-offs today.)

## Commands

| Command | What it does |
|---|---|
| `init [--link <id>]` | create `shutup.config.yaml` + a default `dev` env; inject CLAUDE.md block |
| `set <NAME> [val] [--public] [--env]` | write a value to the env (hidden prompt for secrets) + consume it |
| `use` / `unuse <NAME>` | add/remove a name from this project's consumed set (no value) |
| `missing [--env]` | consumed vars with no value in the env (names to stdout) |
| `list [--env]` | consumed vars + state; secrets never shown |
| `check <NAME> [--env]` | `exists`/`not found` (exit 0/1), never the value |
| `run [--env] -- <cmd>` | inject only consumed vars (least-privilege) and exec |
| `env add/ls/default/rm` | manage which envs a project uses |
| `env export/import` | hand a secret-free env bundle (id + public values + secret names) to a teammate |
| `import <file>` | migrate a `.env` (bare = list names; `--public`/`-i` to classify) |
| `destroy [--yes]` | remove the project (config + CLAUDE.md block); keeps shared envs |

`--env <name>` defaults to the project's `default_env`.

## Using shutup in a project

Wrap your scripts so values are injected at runtime and never written to a committable
`.env`:

```jsonc
// package.json
"scripts": {
  "dev":     "shutup run -- next dev",
  "build":   "shutup run -- next build",
  "migrate": "shutup run -- node scripts/migrate.js"
}
```

**Docker** — public config as build args; secrets injected at *run* time (never baked
into image layers):

```sh
shutup run -- docker build --build-arg NEXT_PUBLIC_APP_URL -t myapp .
shutup run -- docker run --rm -p 3000:3000 \
  -e DATABASE_URL -e STRIPE_SECRET_KEY -e PORT -e NODE_ENV myapp
```

**Monorepo** — each app is its own project (`shutup init` in each). Share an env by
linking the same id (`shutup env add dev --link <id>`, find ids via `shutup env ls --all`),
then `shutup use <NAME>`. Each project's `run` injects only what it consumes.

**Onboarding** — teammate clones (config maps `dev` → the shared env id); you send them a
bundle (`shutup env export dev -o dev.bundle`); they `shutup env import dev.bundle`, then
`shutup missing` shows the secrets to set (their own values — secrets never leave a machine).

## Development

From `cli/`:

```sh
make build     # build to ./shutup
make install   # build + install to ~/.local/bin/shutup
make test      # run tests
make cover     # tests with coverage summary
make vet       # go vet
make fmt       # gofmt -w
```

## Architecture

The `EnvStore` interface (`cli/internal/env`) is the swappable seam — `LocalEnvStore`
today, a cloud `APIStore` or encrypting wrapper later, with no command changes.

```
cli/internal/
  cmd/      cobra commands (agent-legible help)
  config/   shutup.config.yaml: consumes + envs(name→id) + default_env
  env/      Env/Var + EnvStore + LocalEnvStore + secret-free bundle export/import
  project/  ties config + env store: resolve, missing, set (auto-wire), run (only consumed)
  dotenv/   .env parser for `import`
  tty/      /dev/tty hidden + visible prompts (the wow); Unix today, CONIN$ seam for Windows
  agent/    the CLAUDE.md instruction block
  id/       envlocal_/env_ id generation + validation
  ui/       TTY-aware colored output
```

No at-rest encryption yet (`TODO: encrypt` behind `EnvStore`). cgo-free for clean
cross-compilation and distribution later.

## Honest scope

The TTY-bypass protects secret **input** — values never enter an agent's context as you
type. The local store is plaintext on disk in v1 and the agent runs as you, so this is a
strong **guardrail against incidental leakage**, not a sandbox against a hostile agent.
Live multi-person secret sharing, access control, rotation, audit, and packaging are
deferred (see [VISION.md](VISION.md)).
