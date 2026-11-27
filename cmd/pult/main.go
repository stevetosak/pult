// Command pult is the Machine side of Pult. It checks the Action file, lists
// its Actions and runs one of them; the app calls it over SSH (ADR 0002, ADR
// 0004).
package main

import (
	"fmt"
	"os"
)

const usage = `usage:
  pult check [--file <path>]   check the Action file
  pult list [--json]           list the Actions
  pult run <id> [--param <name>=<value>]... [--params <base64url>] [--expect <pin>]
                               run an Action

The Action file is $XDG_CONFIG_HOME/pult/actions.yaml (default
~/.config/pult/actions.yaml).
`

// The exit codes of pult. A Run that starts exits with the command's own code.
const (
	exitOK     = 0
	exitFaults = 1   // check found faults, or list could not read the file
	exitUsage  = 2   // a command or flag that pult does not know
	exitRefuse = 125 // run refused before the command started
	exitExec   = 126 // the command could not be started
)

func main() {
	os.Exit(pult(os.Args[1:], hostSystem()))
}

func pult(args []string, sys *system) int {
	if len(args) == 0 {
		fmt.Fprint(sys.stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "check":
		return check(args[1:], sys)
	case "list":
		return list(args[1:], sys)
	case "run":
		return run(args[1:], sys)
	case "help", "-h", "--help":
		fmt.Fprint(sys.stdout, usage)
		return exitOK
	default:
		fmt.Fprintf(sys.stderr, "pult: unknown command %q\n%s", args[0], usage)
		return exitUsage
	}
}
