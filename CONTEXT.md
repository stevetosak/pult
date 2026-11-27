# pult

Pult is an Android app that runs actions on the Operator's machines with one
tap and opens a terminal on them, over SSH. Each machine describes its own
actions.

## Language

**Operator**:
The person who owns the machines and the phone, and every decision.
_Avoid_: user (in design text)

**Machine**:
A computer that Pult reaches over SSH: a laptop, a Proxmox host, a server.
_Avoid_: host (except for the SSH host key), server (when any machine is meant)

**Action**:
A named command on a Machine, with a title, a group, a Kind and Parameters, defined in that Machine's Action file.
_Avoid_: snippet, shortcut, script

**Action file**:
The YAML file on a Machine that lists its Actions; the `pult` command reads it.
_Avoid_: catalog, config

**Kind**:
How an Action runs. `output` (the default): without a terminal and with no input; the app shows its output and exit code. `terminal`: in a terminal on the phone, for prompts and interactive programs. Both end when the connection ends, unless the command detaches itself.
_Avoid_: background (it suggests the Run outlives the connection)

**Parameter**:
A typed value the Operator gives when a Run starts. It reaches the command as an environment variable, never as text in the command. The history keeps it, so a Parameter is never a password.
_Avoid_: argument, variable (in design text)

**Run**:
One execution of an Action. The history keeps its Machine, Action, Parameters, start, duration and exit code; it never keeps terminal input.

**Action pin**:
The hash of an Action's canonical form that the Operator confirmed, for one Machine. It covers the whole Action, its title too, so any change asks again before the Action runs.
_Avoid_: trust hash

**Host key pin**:
The SSH host key of a Machine that the Operator confirmed on the first connection. A changed key stops the connection.

**Secret field**:
A password field on the phone that sends its text, and Enter, into a terminal Run, and keeps nothing.

**Core**:
The Go logic of the app: Machines, SSH sessions, Runs, history and pins. It is built into the app as an Android library.
_Avoid_: backend

**Bridge**:
The thin Go package that gomobile binds, between the Core and the Kotlin screens. It holds no logic that a Go test cannot reach.

**Feature**:
A tab on a Machine's screen (Actions, Terminal, later Files or Status), a module behind one interface.
_Avoid_: plugin
