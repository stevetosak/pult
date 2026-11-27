---
status: accepted
date: 2026-11-27
---

# Actions live on each Machine and are read over SSH

An Action needs a home: the app could store every Machine's Actions, or each
Machine could describe its own. The Operator chose the second on 2026-11-27.
The app is a remote shell to the Operator's Machines, so where Actions live,
how the app reaches a Machine and what runs without the Operator's look are
one decision.

**Each Machine describes its own Actions.** An Action file on the Machine, in
the user's configuration directory, lists them; the `pult` command reads it.
The Machine knows its own commands, and the file is versioned with the rest of
that Machine's configuration. The app holds no command text of its own: it
holds the Machines, the pins and the history.

**SSH is the only transport: no daemon, no new port.** The app runs
`pult list --json` and `pult run` over SSH. A service that runs commands on
request is a remote shell, and its login check would be one more to get right.
SSH already has one: on a tailnet Machine, Tailscale SSH checks the tailnet
identity and the phone holds no key. A Machine without it gets an SSH key made
in the Android Keystore, which cannot leave the phone.

**Host keys are pinned.** On the first connection the app shows the Machine's
host key fingerprint and pins it once the Operator confirms; a changed key
stops the connection.

**A Parameter reaches the command as an environment variable, never as text in
it.** A value therefore cannot add shell syntax. `pult run` checks every value
against its declared type again; the check in the app's form only helps the
Operator.

**The app pins each Action, and `pult run` checks the pin.** The Core hashes the
canonical form of each Action it lists, with the same `action` package as the
Machine. A new or changed Action shows its command and needs the Operator's
confirmation once. The app passes the pinned hash to `pult run --expect`, which
refuses the Run when the file changed between the list and the Run. The pin
catches an Action that changed without the Operator seeing it, for example an
edit by an agent on the Machine. It does not protect against a process of the
same user that replaces the `pult` binary.

**A terminal Run gets the SSH session's terminal.** `pult run` replaces itself
with the command (exec), so a password prompt reads the terminal directly and no
Pult code is in the path of what is typed. The Secret field sends its text and
Enter into that terminal and keeps nothing; the history of a terminal Run holds
no input.

## Considered options

- **Actions in the app.** One place for every Machine; but the app would hold
  command text for Machines it cannot check, and every change on a Machine
  would need an edit on the phone.
- **A daemon on each Machine with an HTTP API.** Better status and alerts
  pushed to the phone; but it is a new listening service with its own login
  check, and an interactive unlock would pass the password through it.

## Consequences

- Each Machine needs the `pult` binary and an Action file.
- Status and alerts come later as `pult` subcommands over SSH; a daemon is
  considered only when polling over SSH is not enough.
- A Machine the phone cannot reach over SSH has no Actions in Pult.

## Amendment 2026-11-27: the values cross SSH encoded

SSH carries one command string, which the login shell on the Machine runs. A
Parameter value written into that string as `--param name=value` would be read
by that shell first, so `$(…)` in a value would run before `pult` starts. The
Core therefore sends `pult run <id> --expect <hash> --params <base64url>`, where
the last argument is the base64url form of a JSON object of the values. Every
character of the arguments is then in `[A-Za-z0-9_-]`, and no login shell (zsh,
bash, fish, dash) finds syntax in them; a Go test proves the character set. `--param name=value` stays for a person in a terminal on the
Machine, and both forms pass the same checks.

Quoting for the remote shell depends on each Machine's login shell. Values on
stdin would be echoed by a terminal Run's PTY. SSH `env` requests are refused
by Tailscale SSH and sshd unless an `AcceptEnv` rule names them.

## Amendment 2026-11-27: the login to Tailscale SSH, proven from the phone

A spike proved the login before the Core was written. A static linux/arm64
client built on `golang.org/x/crypto/ssh` v0.59.0 ran on the phone through
`adb` and logged in to the laptop's Tailscale SSH (Tailscale 1.104.7) over the
tailnet; two runs gave the same results. Its code stays on the branch
`spike/tailscale-ssh` and is not merged.

**No key is needed.** With no authentication method, the client sends only
`none`, and Tailscale SSH accepts it for a tailnet identity its SSH rules
allow, in about 300 ms. A command runs in the user's login shell; its exit
status (0, and 125 for a refused pin) and its stdout and stderr arrive apart.
A PTY session gets the size the client asks for, and a window change reaches
it: the terminal reports the new size and the process gets `SIGWINCH`.

**A rule in check mode holds the login until the Operator approves it in a
browser.** During authentication the server sends one banner, "Tailscale SSH
requires an additional check" with a `login.tailscale.com` URL, and then
nothing until the check passes; it sets no time limit of its own. Logins after
the approval send no banner until the rule's check period ends. The Core
therefore shows each banner as it arrives, with its URL ready to open, and
waits until the Operator cancels: a time limit on authentication would fail
every login that needs a check.

**The Core chooses the host key algorithm.** Tailscale SSH offers an Ed25519,
an ECDSA P-256 and an RSA key. With its defaults, `x/crypto/ssh` negotiated the
ECDSA key, so a pin would follow the library's preference order and break when
that order changes. The Core sets the host key algorithm itself and pins that
key; which algorithm is decided with the Core. On the Machine, `ssh-keygen -lf`
reads Tailscale's ECDSA and RSA key files but not its Ed25519 file, which
OpenSSL reads: that fingerprint comes from `openssl pkey -pubout`, hashed in the
SSH wire form. All three fingerprints the spike received matched the keys on
the Machine.

**The tunnel can go away under a session.** Android stops an app's VPN when the
app is updated, and Tailscale stays off until it is opened again, unless it is
the always-on VPN. A connection open at that moment fails with "software caused
connection abort". The Core reports it as a lost connection, not as a refused
login.
