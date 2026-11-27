---
title: "First version: Actions and a terminal from the phone"
description: "The first version of Pult: the Action file and the pult command on a Machine, the Go Core with its Bridge, the Android app with one-tap Actions and a terminal, and a signed release the Operator installs."
type: plan
status: active
topics: [android, remote-access]
---

Remote access from the phone was a terminal app and typed commands. Typing a
long command, and a password prompt that opened on the laptop's screen instead
of the phone's, made it slow and error-prone. Pult replaces that with Actions:
each Machine lists its own in an Action file, and the app runs any of them with
one tap
([ADR 0002](../adr/0002-actions-live-on-each-machine-and-are-read-over-ssh.md)).
The app is a Go Core behind Kotlin and Compose screens, Android only
([ADR 0001](../adr/0001-a-go-core-behind-kotlin-and-compose-screens.md)).

**The first version is finished when the Operator, away from the laptop,
unlocks its password vault, keeps it awake and opens a shell on it from Pult,
without typing a command.** Everything after that (status on a Machine's card,
pinned Actions on the home screen, files, port forwards, alerts, tsnet, more
Machines) becomes its own Plan.

Two risks come first, because each could change the design: the Action file
format, which both sides share, and the login to Tailscale SSH from Go with no
key. The first part and the spike of the second prove them before any screen
is built.

## The Action file and the pult command

**Done when:** on the laptop, `pult check` accepts an Action file with the first
Actions and refuses a broken one with the line of the fault; `pult list --json`
lists them; `pult run keep-awake --param hours=1` holds a sleep inhibitor that
`systemd-inhibit --list` shows; `pult run unlock-vault` in a terminal unlocks
the password vault; and a Run with a wrong `--expect` hash is refused before
anything starts.

- [x] Decide in a grill the Action file format v1: the fields (id, title, group,
      Kind, confirm, Parameters, run), the Parameter types and their limits, the
      environment names a Parameter gets, the canonical form that a pin
      covers, where the file lives, and the first Actions (vault unlock, vault
      status, keep awake, lock screen, a Claude Code Remote Control session):
      strict YAML at `~/.config/pult/actions.yaml`, the Kinds `output` and
      `terminal`, Parameters as `PULT_<NAME>`, values base64url-encoded across
      SSH, and a pin over the whole Action
      ([ADR 0003](../adr/0003-the-action-file-is-strict-yaml-and-a-pin-covers-the-whole-action.md),
      ADR 0002 amended).
- [x] The Go module and the `action` package: parse, validate with the line of
      each fault, the canonical form and its hash, Parameters to environment,
      and the base64url encoding of the values. Choose the YAML library
      (`yaml/go-yaml` or `goccy/go-yaml`) by its line positions and strict
      decoding. Table tests, with Parameter values that hold shell syntax;
      golden tests of the canonical form; a test that the encoded command holds
      only the safe characters: `go.yaml.in/yaml/v4`, Go 1.27.4, one validator
      for the file and for JSON, and the command passed through sh, dash, bash
      and zsh unchanged
      ([ADR 0003](../adr/0003-the-action-file-is-strict-yaml-and-a-pin-covers-the-whole-action.md),
      amendment).
- [ ] `cmd/pult`: `check` (the file permissions, `sh -n` on each `run`, and a
      `$PULT_` name that no Parameter declares or a Parameter that `run` never
      uses), `list --json` and
      `run <id> [--param name=value]… [--params <base64url>] [--expect <hash>]`.
      `run` checks the values and the pin, then replaces itself with `sh -c`
      and the environment, with its input from `/dev/null` for an `output`
      Action.
- [ ] Install `pult` on the laptop with its first Action file, and run each
      Action from a terminal there: `unlock-vault` (terminal), `vault-status`
      and `lock-vault`, each with a `vault` choice; `keep-awake` (`hours` 1 to
      12, and `lid`, default false, which adds the `handle-lid-switch`
      inhibitor) and `allow-sleep`; `lock-screen` (`loginctl lock-session` of
      the graphical session, which needs no admin authentication, unlike
      `lock-sessions`); `claude-session` (a `project` choice, in a new
      terminal workspace tab). First read the PATH that a Tailscale SSH command
      gets. The repository gets `examples/actions.yaml` with generic values.

## The Core and the Bridge

**Done when:** `go test ./...` passes; `gomobile bind` builds the Android
library with the pinned toolchain; and a Go test client on the laptop, with no
SSH key, logs in to a tailnet Machine through Tailscale SSH, pins its host key,
lists its Actions, runs keep-awake with a Parameter and streams its output, and
runs unlock-vault in a terminal session.

- [ ] Spike: `golang.org/x/crypto/ssh` logs in to Tailscale SSH with no key
      (the client's `none` method) and reads the host key the server shows.
- [ ] The Android toolchain on the laptop, at versions read from their sources
      at install time and pinned: the command-line tools, the SDK platform, the
      NDK and gomobile. A `make` target binds the Core.
- [ ] Core: the Machine store, host key pins, SSH sessions (a command with its
      output streams and exit code; a terminal with resize), Runs with their
      history, Action pins, and the `pult run` command with the values encoded
      (ADR 0002 amendment).
- [ ] Bridge: decide its API (JSON values, callback interfaces for output and
      terminal bytes, errors), then build it with a panic guard and tests.

## The Android app

**Done when:** on the phone, over the tailnet, the app opens only after a
fingerprint; the Operator adds the laptop and confirms its host key;
unlock-vault with the Secret field unlocks the vault, which then reads as
unlocked on the laptop; keep-awake with a Parameter shows its output and exit
code; the Terminal tab opens a shell; and an Action changed in the file asks
again before it runs.

- [ ] Decide in a grill the screens and the navigation, from the sketches of
      2026-11-27 (the Machines, a Machine with its Feature tabs, a Run with its
      Parameter form and output, a terminal Run), the Feature interface and
      the theme.
- [ ] The Gradle project at pinned versions: Kotlin, Compose, Material 3, the
      Core's library, the fingerprint lock (BiometricPrompt), and screenshots
      blocked on the terminal and Secret field screens (`FLAG_SECURE`).
- [ ] Machines: the list, adding a Machine, the host key confirmation.
- [ ] The Actions Feature: groups, the Parameter form, the confirm step, the pin
      dialog for a new or changed Action, the output sheet, the history.
- [ ] The Terminal Feature: connectbot/termlib, a row of extra keys (Esc, Tab,
      Ctrl, arrows) and the Secret field; terminal Runs and a shell.

## Build and install

**Done when:** CI runs the Go tests, binds the Core, and builds and tests the
app on every pull request; a tag makes a GitHub Release with an APK signed with
the Operator's key; and the Operator installs it on the phone and the next
release installs over it.

- [ ] CI with the pinned toolchain: the Go tests, the bind, the Gradle build and
      its tests.
- [ ] The Operator creates the release signing key and stores it; CI signs the
      release APK with it from a repository secret.
- [ ] A release on each tag; install it on the phone, then install the next one
      over it.
