---
status: accepted
date: 2026-11-27
---

# pult refuses before a Run starts, and then replaces itself with sh

The `pult` command is the Machine side of Pult: the app runs `pult list --json`
and `pult run` over SSH, and the Operator runs `pult check` after an edit
([ADR 0002](0002-actions-live-on-each-machine-and-are-read-over-ssh.md),
[ADR 0003](0003-the-action-file-is-strict-yaml-and-a-pin-covers-the-whole-action.md)).
The Operator let the agent decide its details and reviewed the result
(2026-11-27).

## Refusals and exit codes

**`pult run` checks everything before the command starts, and refuses with exit
code 125.** It refuses when the Action file cannot be used, the id is unknown,
the pin is not the one given with `--expect`, or a value is missing, undeclared
or outside its limits; the message starts with `pult: refused:`. 125 is the
code that `env`, `chroot` and container runtimes use for a failure of their
own, so the app tells a refusal from a command's exit code. 126 means that
`/bin/sh` could not be started. A Run that starts exits with the command's own
code, because `pult` is no longer there.

**A file with faults refuses every Run, not only the broken Action.** With an
id used twice or an unknown version, which Action a Run means is not clear, and
`pult check` names every fault. `pult list` refuses the same way, with exit
code 1.

**The Action file must belong to the user.** The file, its directory, and for a
symbolic link also its target and the target's directory, must be owned by the
user or root and not be writable by group or others. The target counts because
an Action file kept in a dotfiles repository is often a link, and whoever can
write that repository's directory can replace the file.

## pult check

`pult check [--file <path>]` reports every fault as `path:line:column: message`
and exits 1 when there is one. Beyond the rules of the format it runs `sh -n`
on each `run`, with the Action's id as `$0`, so the shell's own message names
the Action and the line in the command. A `$PULT_` name that no Parameter
declares is a fault: the environment of a Run drops inherited `PULT_`
variables, so the name is always empty. A Parameter that `run` never names is
only a warning, because `run` can start a script that reads the variable
itself. `--file` checks a draft before it replaces the real file; `list` and
`run` read only the real file.

## pult list

`pult list` prints a table of id, Kind, group and title. `pult list --json`
prints the Listing, `{"version":1,"actions":[…]}`, with each Action under its
YAML field names and no pins, since the Core computes each pin from the Action
it shows. The Core reads it with `action.DecodeListing`, which refuses an
unknown field: a newer `pult` can add a field that its pins cover, and an older
Core that ignored the field would compute other pins and see every Run refused.

## pult run

`pult run <id> [--param name=value]… [--params <base64url>] [--expect <pin>]`
takes the id first. `--param` is for a person in a terminal and `--params` for
the Core; values from both are joined, and a name given twice is refused.
`--expect` is optional for a person and always sent by the Core. After its
checks, `pult` changes to the home directory, puts `/dev/null` on the input of
an `output` Action, and replaces itself with
`/bin/sh -c <run> <id>` and the environment of the SSH session less every
`PULT_` variable plus the Parameters.

## Libraries

The command line uses the standard library's `flag`; the id is taken before the
flags because `flag` stops at the first argument that is not a flag. The exec
and the copy of `/dev/null` onto the input use `golang.org/x/sys/unix`, which
has `Dup2` on every Unix system; the standard library's `syscall.Dup2` is
missing on Linux on arm64.

## Considered options

- **cobra or urfave/cli** for the command line. Help, completion and nested
  commands come free, but there are three commands and four flags, and each
  dependency is code in a binary that runs on every Machine.
- **Run a good Action from a file with faults elsewhere.** Convenient after a
  bad edit, but an id used twice would make the Run ambiguous.
- **An unused Parameter as a fault.** It would refuse an Action whose script
  reads the variable itself.
- **Every directory up to the root checked, as sshd's `StrictModes` does.**
  ADR 0003 decided on the file and its directory; a home directory that others
  can write is a larger fault than Pult can repair.
- **Exit code 1 or 2 for a refusal.** Commands use both for their own faults,
  so the app could not tell them apart.

## Consequences

- The app reads exit code 125 and the `pult: refused:` line as a refusal, and
  any other code as the command's own.
- A Machine that the Core lists with a newer `pult` than the app knows shows an
  error instead of pins that never match.
