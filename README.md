# Pult

Pult is an Android app for remote access to your own machines. Each machine
describes the actions it offers (unlock a password vault, keep the machine
awake, start a session) in a small action file; the app lists them and runs
any of them with one tap, over SSH. An action that asks for input, such as a
password prompt, runs in a real terminal on the phone.

The name is the Macedonian *пулт*: a control desk.

## Status

Planning. Nothing is built yet.

- [The first version](docs/plans/first-version-actions-and-a-terminal-from-the-phone.md): the Plan.
- [A Go core behind Kotlin and Compose screens](docs/adr/0001-a-go-core-behind-kotlin-and-compose-screens.md): the stack.
- [Actions live on each machine and are read over SSH](docs/adr/0002-actions-live-on-each-machine-and-are-read-over-ssh.md): the machine side.
- [The Action file is strict YAML, and a pin covers the whole Action](docs/adr/0003-the-action-file-is-strict-yaml-and-a-pin-covers-the-whole-action.md): the Action file format v1.
- [CONTEXT.md](CONTEXT.md): the words this project uses.
