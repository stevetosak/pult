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
