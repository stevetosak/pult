package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/stevetosak/pult/action"
)

const runUsage = "pult run <id> [--param <name>=<value>]... [--params <base64url>] [--expect <pin>]"

// run starts one Action. It refuses before anything starts, with exit code
// 125, when the file has faults, the Action does not exist, its pin is not the
// one given with --expect, or a value is missing, undeclared or outside its
// limits. It then replaces itself with sh -c and the Action's run, in the
// home directory, with the environment of the SSH session less any inherited
// PULT_ variable plus the Parameters; an output Action's input is /dev/null.
func run(args []string, sys *system) int {
	refuse := func(format string, a ...any) int {
		fmt.Fprintf(sys.stderr, "pult: refused: "+format+"\n", a...)
		return exitRefuse
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return refuse("the Action id comes first: %s", runUsage)
	}
	id := args[0]
	flags := flag.NewFlagSet("pult run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var params paramFlags
	flags.Var(&params, "param", "")
	encoded := flags.String("params", "", "")
	expect := flags.String("expect", "", "")
	if err := flags.Parse(args[1:]); err != nil {
		return refuse("%v: %s", err, runUsage)
	}
	if flags.NArg() > 0 {
		return refuse("unexpected argument %q: %s", flags.Arg(0), runUsage)
	}
	given, err := values(params, *encoded)
	if err != nil {
		return refuse("%v", err)
	}

	path, err := actionFile(sys)
	if err != nil {
		return refuse("%v", err)
	}
	f, err := loadFile(path, sys)
	if err != nil {
		printFaults(sys.stderr, path, err, "")
		return refuse("pult cannot use the Action file; pult check shows every fault")
	}
	a, found := f.Action(id)
	if !found {
		return refuse("%s has no Action %q", path, id)
	}
	if *expect != "" {
		if pin := action.Pin(a); pin != *expect {
			return refuse("the Action %s changed since it was pinned: its pin is %s, not %s", id, pin, *expect)
		}
	}
	bindings, err := a.Bind(given)
	if err != nil {
		return refuse("the values of %s:\n  %s", id, strings.ReplaceAll(err.Error(), "\n", "\n  "))
	}
	home, err := homeDir(sys)
	if err != nil {
		return refuse("%v", err)
	}

	err = sys.exec(command{
		path:    shell,
		argv:    []string{"sh", "-c", a.Run, a.ID},
		env:     action.Environ(sys.environ(), bindings),
		dir:     home,
		noInput: a.EffectiveKind() == action.KindOutput,
	})
	if err != nil {
		fmt.Fprintf(sys.stderr, "pult: cannot start %s: %v\n", shell, err)
		return exitExec
	}
	return exitOK // only a test's exec returns without an error
}

// values joins the values of --params and of each --param. A name given
// twice is refused.
func values(params paramFlags, encoded string) (map[string]string, error) {
	given := map[string]string{}
	if encoded != "" {
		decoded, err := action.DecodeValues(encoded)
		if err != nil {
			return nil, err
		}
		given = decoded
	}
	for _, p := range params {
		if _, twice := given[p.name]; twice {
			return nil, fmt.Errorf("a value for %s is given twice", p.name)
		}
		given[p.name] = p.value
	}
	return given, nil
}

// paramFlags collects each --param name=value.
type paramFlags []struct{ name, value string }

func (p *paramFlags) String() string { return "" }

func (p *paramFlags) Set(s string) error {
	name, value, found := strings.Cut(s, "=")
	if !found || name == "" {
		return errors.New("--param takes name=value")
	}
	*p = append(*p, struct{ name, value string }{name, value})
	return nil
}
