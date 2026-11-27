---
status: accepted
date: 2026-11-27
---

# The Action file is strict YAML, and a pin covers the whole Action

The Action file is the one contract between the two sides of Pult: the `pult`
command on a Machine reads it, and the app's Core reads what `pult list --json`
makes of it, both through the `action` package
([ADR 0002](0002-actions-live-on-each-machine-and-are-read-over-ssh.md)). A
change to its format after the first Machines use it breaks every file and every
Action pin, so version 1 was decided in a grill on 2026-11-27, before any code.

## The file

**The Action file is strict YAML.** Commands are often several lines long and
Parameters are nested; a YAML block scalar reads better for both than TOML's
multi-line strings and arrays of tables, and the Operator reads YAML every day.
The file is decoded strictly into typed fields, and an unknown field is a fault,
which stops YAML's implicit-type traps (`no`, `0755`, `1.10`). The parser is a
library, chosen in the build step: `gopkg.in/yaml.v3` has been archived since
2025-04, while its successor `yaml/go-yaml` and `goccy/go-yaml` are active.

**There is one file, in the user's configuration directory:**
`$XDG_CONFIG_HOME/pult/actions.yaml`, or `~/.config/pult/actions.yaml` when the
variable is not set. `pult` takes `--file` only for `check` and tests, and there
is no include and no directory of files in v1. The top level is `version: 1`
and an `actions` list, whose order is the order on the screen.

**A file that others can write is refused.** `pult` refuses the Action file, or
its directory, when a group or other users can write it, as `sshd` does with
`authorized_keys`.

```yaml
version: 1
actions:
  - id: unlock-vault
    title: Unlock the vault
    group: Vault
    kind: terminal
    params:
      - name: vault
        title: Vault
        type: choice
        options: [personal, work]
        default: work
    run: RBW_PROFILE="$PULT_VAULT" exec rbw unlock

  - id: keep-awake
    title: Keep awake
    group: Session
    params:
      - name: hours
        title: Hours
        type: int
        min: 1
        max: 12
        default: 2
      - name: lid
        title: Also with the lid closed
        type: bool
        default: false
    run: |
      what=sleep:idle
      if [ "$PULT_LID" = true ]; then what="$what:handle-lid-switch"; fi
      setsid -f systemd-inhibit --what="$what" --who=pult --why="Pult keep-awake" \
        sleep "${PULT_HOURS}h" >/dev/null 2>&1
```

## The Action

- `id` (required): `^[a-z][a-z0-9-]{0,47}$`, unique in the file. `pult run`,
  the pin and the history key on it; a new id is a new Action, and the old
  history stays under the old id.
- `title` (required, one line, at most 60 characters), `description` (optional,
  at most 200) and `group` (optional, at most 30). Groups take the order in
  which they first occur; Actions without a group come first.
- `kind`: `output` (the default) or `terminal`.
- `confirm` (default `false`): the app shows the title and the description and
  asks before the Run starts.
- `params`: the Parameters, below.
- `run` (required, at most 4 KiB): one string, which `pult run` starts with
  `sh -c` (POSIX `sh`, dash on Ubuntu) in the home directory, with the SSH
  session's environment and the Parameters. Pipes, `&&`, `env -u` and
  `setsid -f` work as written, and the Parameters are already environment
  variables, so no value becomes shell code. There are no `shell`, `cwd`,
  `timeout` or `env` fields in v1: bash is `bash -c '…'` or a script, and the
  app's Stop is the timeout.

**An `output` Run has no terminal and no input.** Its input is `/dev/null`, so a
command that prompts fails at once instead of waiting, and the app shows its
output and exit code. A `terminal` Run gets the SSH session's terminal.

**A Run ends with its connection; a command that must outlive it detaches
itself.** When the session closes, the command gets SIGHUP, and the app's Stop
closes the session. Keep-awake and a Claude Code Remote Control session must
outlive the phone's connection, so their commands detach (`setsid -f`, a
terminal workspace tab). The detach stays inside the SSH session's login
session: a sleep inhibitor needs an active session under logind's polkit rule
(`inhibit-block-sleep` allows `active`, not `any`), and Ubuntu's logind keeps a
session's processes after the logout (`KillUserProcesses=no`).

**`pult check` runs `sh -n` on each `run`:** the syntax check of the shell that
runs it, so a broken command is found on the Machine before the phone tries it.

## Parameters

**Parameters are a list under `params`, in the order of the form.** Each has a
`name`, a `title`, a `type`, an optional `default` and the limits of its type. A
Parameter without a default must be given a value, so there is no `required`
field. An Action has at most 8 Parameters.

| Type | Limits | Environment value |
|---|---|---|
| `string` | `max` length (default 200, at most 1024); optional `pattern`, a Go RE2 expression matched against the whole value | the text |
| `int` | `min` and `max`, both required | decimal |
| `bool` | — | `true` or `false` |
| `choice` | `options`, a list of strings | the option |

No type accepts a control character (newline, ESC, NUL). There is no `secret`
type, because the history keeps every Parameter: a password goes through the
Secret field into a terminal Run.

**A Parameter reaches the command as `PULT_` and its name in capitals**
(`hours` becomes `$PULT_HOURS`); a name matches `^[a-z][a-z0-9_]{0,31}$`. The
prefix makes the rule safe by construction: a Parameter cannot replace `PATH`,
`LD_PRELOAD`, `BASH_ENV` or `IFS`. It also shows in `run` which values come from
Pult, and lets `pult check` report a `$PULT_` name that no Parameter declares
and a Parameter that `run` never uses.

**`pult run` refuses before anything starts** a Parameter the Action does not
declare, a missing value and a value outside its limits. The app's form checks
the same limits only to help the Operator. How the values cross SSH is an
amendment to [ADR 0002](0002-actions-live-on-each-machine-and-are-read-over-ssh.md).

## The Action pin

**An Action pin covers the whole Action.** Every field counts, the title too:
otherwise two pinned Actions could swap titles, and "Lock the screen" would run
the other command without a question. A typo fix then asks again, which costs
one tap. The rule on the screen is: anything changed, ask again.

**The canonical form is JSON of the decoded Action, not the bytes of the
file.** `pult run` hashes the Action from the YAML file and the Core hashes the
Action it received from `pult list --json`, so comments and spacing in the file
never change a pin. The fields are in a fixed order with no spaces, and **a
field at its default value is left out**: `kind: output` written or omitted
gives the same pin, and a field that a later version adds leaves the pins of
older Actions as they were. The order of `params` and of `options` counts,
because it is the order on the screen.

**The form is made by own code and held by golden tests.** `encoding/json` of
one fixed canonical struct, with no maps and only strings, integers and
booleans, has fully defined output. Golden tests (a fixed Action gives a fixed
hash, for all time) catch any drift between versions, which matters because the
`pult` on a Machine and the app's Core can be different versions.

**The pin string is `v1-` and the SHA-256 in lowercase hex** (67 characters),
inside the character set the SSH hop allows. `v1` names the canonical form, so a
later form can exist beside it. A pin belongs to one Machine and one id. The
Core computes it from the Action it shows the Operator, never from a hash that
`pult` sends.

## Considered options

- **TOML.** It has no implicit types, and `BurntSushi/toml` and
  `pelletier/go-toml` are maintained; but nested Parameters become arrays of
  tables that read worse, and strict YAML decoding removes the main trap.
- **A map keyed by id** instead of a list. Uniqueness would be structural, but
  the order on the screen would depend on the parser.
- **The Kind name `background`.** It suggests that the Run outlives the
  connection, which no Kind does.
- **A `detached` Kind in v1**, which `pult` starts with `setsid`, logs and
  tracks with `pult ps` and `pult stop`. It shows what runs and stops it, but it
  needs a state directory, PID reuse handling and log cleanup before it is clear
  which Actions need it. It, or `status` and `stop` fields that make an Action a
  switch, is left to a later Plan; nothing in v1 blocks either.
- **`run` as an argv list**, or both forms. No quoting trap, but a pipe or a
  detach needs `sh -c` inside the list anyway, and two forms are two paths to
  test and to hash.
- **`confirm` as a message.** The description already carries a warning, and
  one type decodes strictly.
- **A shell parser library (`mvdan.cc/sh`) in `pult check`.** It could find an
  unquoted `$PULT_` name, but its POSIX mode can disagree with dash; it is left
  for later.
- **Parameter names as environment names, with a list of refused names.**
  Shorter, but such a list is never complete: `BASH_ENV` alone runs a file when
  bash starts.
- **A pin over only the fields that change the Run** (`id`, `kind`, `run`,
  `params`, `confirm`). Text edits would be free, but a title swap would pass
  without a question.
- **An RFC 8785 (JCS) library** (`cyberphone/json-canonicalization`,
  `gowebpki/jcs`) for the canonical form. It solves map ordering and number
  formatting, which a fixed struct does not have, and the golden tests are
  needed anyway.

## Consequences

- The `action` package holds the strict decoding with the line of each fault,
  the limits, the canonical form and its golden tests, and the encoding of
  Parameter values for the SSH hop.
- A change that alters a golden hash is a breaking change: it needs a new
  canonical form beside `v1`, never an edit of `v1`.
- An Action that must outlive the connection carries its own detach until a
  later Plan gives Pult one.
- The repository's example file uses generic values; each Machine's own file
  stays on that Machine.
