package action

import (
	"fmt"
	"regexp"
)

// pultReference finds a PULT_ variable in a command: $PULT_NAME or
// ${PULT_NAME...}.
var pultReference = regexp.MustCompile(`\$\{?PULT_([A-Za-z0-9_]*)`)

// Check returns what pult check finds beyond the rules that Parse applies,
// with the position of each finding in the file:
//
//   - faults: an Action whose run syntax refuses (pult passes sh -n; nil
//     skips it), and a $PULT_ name that no Parameter declares, which is
//     always empty in the command because the environment of a Run drops
//     inherited PULT_ variables;
//   - warnings: a Parameter whose variable run never names. It can still be
//     right, when run starts a script that reads the variable itself.
func (f *File) Check(syntax func(a Action) error) (faults, warnings Faults) {
	at := func(p path, format string, args ...any) Fault {
		fault := Fault{Path: p.String(), Message: fmt.Sprintf(format, args...), at: p}
		if f.root != nil {
			n := locate(f.root, p)
			fault.Line, fault.Column = n.Line, n.Column
		}
		return fault
	}
	for i, a := range f.Actions {
		run := path{"actions", i, "run"}
		if syntax != nil {
			if err := syntax(a); err != nil {
				faults = append(faults, at(run, "run has a syntax fault: %v", err))
			}
		}
		declared := map[string]bool{}
		for _, p := range a.Params {
			declared[EnvName(p.Name)] = true
		}
		used := map[string]bool{}
		for _, m := range pultReference.FindAllStringSubmatch(a.Run, -1) {
			name := envPrefix + m[1]
			if !declared[name] && !used[name] {
				faults = append(faults, at(run, "run uses $%s, but no Parameter of %s declares it", name, a.ID))
			}
			used[name] = true
		}
		for j, p := range a.Params {
			if !used[EnvName(p.Name)] {
				warnings = append(warnings, at(path{"actions", i, "params", j, "name"},
					"run never names $%s, the variable of Parameter %s", EnvName(p.Name), p.Name))
			}
		}
	}
	faults.sortByPosition()
	warnings.sortByPosition()
	return faults, warnings
}
