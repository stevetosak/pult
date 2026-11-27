# Pult

Pult is an Android app for remote access to your own machines. Each machine
describes the actions it offers (unlock a password vault, keep the machine
awake, start a session) in a small action file; the app lists them and runs
any of them with one tap, over SSH. An action that asks for input, such as a
password prompt, runs in a real terminal on the phone.

The name is the Macedonian *пулт*: a control desk.

## Status

Being built. The `pult` command for each machine (`check`, `list`, `run`) and
the `action` package it shares with the app exist; the app does not yet.

- [The first version](docs/plans/first-version-actions-and-a-terminal-from-the-phone.md): the Plan.
- [A Go core behind Kotlin and Compose screens](docs/adr/0001-a-go-core-behind-kotlin-and-compose-screens.md): the stack.
- [Actions live on each machine and are read over SSH](docs/adr/0002-actions-live-on-each-machine-and-are-read-over-ssh.md): the machine side.
- [The Action file is strict YAML, and a pin covers the whole Action](docs/adr/0003-the-action-file-is-strict-yaml-and-a-pin-covers-the-whole-action.md): the Action file format v1.
- [pult refuses before a Run starts, and then replaces itself with sh](docs/adr/0004-pult-refuses-before-a-run-starts-and-then-replaces-itself-with-sh.md): the `pult` command.
- [CONTEXT.md](CONTEXT.md): the words this project uses.

## The pult command on a machine

`make install` builds `pult` from the checkout into `~/.local/bin` (`PREFIX`
changes the place). There are no release binaries yet.

The app runs `pult` over SSH as a command. The SSH server starts it with the
user's shell as `$SHELL -c`, which is not a login shell and reads neither
`~/.profile` nor `~/.zprofile`; Tailscale SSH also gives that shell a fixed
default `PATH`. `pult`, and each tool an action calls, must therefore be found
on the `PATH` of a non-login `$SHELL -c`. For zsh, every shell reads
`~/.zshenv`:

```sh
typeset -U path; path=("$HOME/.local/bin" $path)
```

Another shell needs the same line in the file that its non-login `-c` form
reads, or `pult` in a directory of the default `PATH`, such as
`/usr/local/bin`.

The action file is `~/.config/pult/actions.yaml`, with mode 0600 in a
directory with mode 0700; `pult` refuses a file that others can change.
[examples/actions.yaml](examples/actions.yaml) is a template with the first
actions: unlock, check and lock an rbw vault, keep the machine awake and allow
sleep again, lock the screen, and start a Claude Code session with Remote
Control in a herdr tab. `pult check` names each fault with its line, and
`pult list` shows what the app will list.

## Building the Core for Android

`make bind` builds the Go Core into an Android library, `build/core.aar`, for
the phone (arm64) and the emulator (amd64). It needs the Android SDK in
`$ANDROID_HOME` (default `~/Android/Sdk`) and a JDK with `javac`. The SDK
packages are pinned in [toolchain.env](toolchain.env), and gomobile and gobind
in `go.mod`.

`make toolchain-check` compares what is installed with the pins and prints the
install command for each difference. The first step installs the command-line
tools by hand: download the `cmdline-tools;latest` archive that
[Google's SDK repository](https://dl.google.com/android/repository/repository2-3.xml)
lists for the pinned version, check its SHA-1 against the repository, and
unpack its `cmdline-tools` directory as `$ANDROID_HOME/cmdline-tools/latest`.
`sdkmanager` then installs the platform and the NDK and checks each download.
