# pult — agent guide

## Repo purpose

Pult is an Android app for remote access to the Operator's machines: one-tap
Actions read from each Machine's Action file, and a terminal, over SSH. The
words are defined in [CONTEXT.md](CONTEXT.md); use them.

## Layout (planned)

```
go.mod                  module github.com/stevetosak/pult; gomobile and gobind pinned as tools
toolchain.env           the Android SDK packages pinned (KEY=value); make toolchain-check, make bind
scripts/                build helpers the Makefile runs
action/                 the Action file format: parse, validate, canonical hash, Parameters to environment
cmd/pult/               the machine command: check, list, run
core/                   the app's Go logic: Machines, SSH sessions, Runs, history, pins
mobile/                 the Bridge that gomobile binds; thin, JSON in and out, callback interfaces
android/                the Gradle project: Kotlin, Jetpack Compose, Material 3
docs/adr/               decisions
docs/plans/             Plans (kalfa reads docs/plans/*.md)
```

## Rules

- The `action` package is the only definition of the Action file format. The
  machine command and the Core both import it; Kotlin never parses an Action
  file or computes a pin.
- The Bridge holds no logic that a Go test cannot reach. Structured values cross
  it as JSON; streams come back through interfaces that Kotlin implements. The
  Bridge recovers a panic and returns it as an error.
- A Parameter reaches a command only as an environment variable. Never build
  command text from a value: the Core sends values to `pult run` only as
  `--params <base64url>` (ADR 0002, amendment).
- The canonical form of an Action is held by golden tests. A change that alters
  a golden hash is a breaking change: it needs a new canonical form beside
  `v1`, never an edit of `v1` (ADR 0003).
- `pult run` replaces itself with the command (exec). No Pult code reads or
  copies what is typed into a terminal Run, and nothing stores the Secret
  field's text or terminal input.
- Pin every toolchain and dependency version (Go, gomobile, Android SDK, NDK,
  Gradle, the Android Gradle Plugin, connectbot/termlib).
- The repository is public: no machine names, addresses, tailnet names,
  account names or working notes in commits, docs or tests. Tests use example
  values.
- The Operator creates every secret value (the release signing key included).
  An agent never reads, generates or prints one.

## Docs

Decisions go in `docs/adr/NNNN-<title>.md` (front matter `status`, `date`; the
title is the decision). Plans go in `docs/plans/` with a `**Done when:**` line
for each part; `kalfa plan check` must pass. Docs are impersonal narrative
prose.
