package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"text/tabwriter"

	"github.com/stevetosak/pult/action"
)

// list prints the Actions in the order of the file: as a table, or with
// --json as the Listing that the app's Core decodes.
func list(args []string, sys *system) int {
	flags := flag.NewFlagSet("pult list", flag.ContinueOnError)
	flags.SetOutput(sys.stderr)
	asJSON := flags.Bool("json", false, "print the Listing as JSON, for the app")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(sys.stderr, "pult list: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	path, err := actionFile(sys)
	if err != nil {
		fmt.Fprintf(sys.stderr, "pult list: %v\n", err)
		return exitFaults
	}
	f, err := loadFile(path, sys)
	if err != nil {
		printFaults(sys.stderr, path, err, "")
		fmt.Fprintln(sys.stderr, "pult list: pult cannot use the Action file; pult check shows every fault")
		return exitFaults
	}
	if *asJSON {
		enc := json.NewEncoder(sys.stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(action.NewListing(f)); err != nil {
			fmt.Fprintf(sys.stderr, "pult list: %v\n", err)
			return exitFaults
		}
		return exitOK
	}
	table := tabwriter.NewWriter(sys.stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tKIND\tGROUP\tTITLE")
	for _, a := range f.Actions {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", a.ID, a.EffectiveKind(), a.Group, a.Title)
	}
	if err := table.Flush(); err != nil {
		return exitFaults
	}
	return exitOK
}
