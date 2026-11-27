package main

import (
	"flag"
	"fmt"

	"github.com/stevetosak/pult/action"
)

// check reports every fault of the Action file with its line: the ownership,
// the format, sh -n on each run and a $PULT_ name that no Parameter declares;
// and, as a warning, a Parameter that run never names.
func check(args []string, sys *system) int {
	flags := flag.NewFlagSet("pult check", flag.ContinueOnError)
	flags.SetOutput(sys.stderr)
	file := flags.String("file", "", "the Action file to check (default: the one pult reads)")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(sys.stderr, "pult check: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	path := *file
	if path == "" {
		var err error
		if path, err = actionFile(sys); err != nil {
			fmt.Fprintf(sys.stderr, "pult check: %v\n", err)
			return exitFaults
		}
	}
	f, err := loadFile(path, sys)
	if err != nil {
		printFaults(sys.stderr, path, err, "")
		return exitFaults
	}
	faults, warnings := f.Check(func(a action.Action) error { return sys.syntax(a.Run, a.ID) })
	if len(faults) > 0 {
		printFaults(sys.stderr, path, faults, "")
	}
	if len(warnings) > 0 {
		printFaults(sys.stderr, path, warnings, "warning: ")
	}
	if len(faults) > 0 {
		return exitFaults
	}
	fmt.Fprintf(sys.stdout, "%s: %d Actions, no faults, %d warnings\n", path, len(f.Actions), len(warnings))
	return exitOK
}
