package action

import (
	"slices"
	"strings"
)

// envPrefix starts the environment name of every Parameter, so that no
// Parameter can replace PATH, LD_PRELOAD, BASH_ENV or IFS.
const envPrefix = "PULT_"

// EnvName returns the environment variable that carries a Parameter:
// PULT_ and the name in capitals.
func EnvName(param string) string { return envPrefix + strings.ToUpper(param) }

// Binding is the checked value of one Parameter, in its environment form.
type Binding struct {
	Name  string
	Value string
}

// Bind checks the values given for a Run of a valid Action and returns one
// Binding for each Parameter, in the order of the Parameters, with the
// defaults filled in. It refuses a value for a Parameter that the Action does
// not declare, a missing value and a value outside its limits, and reports
// all of them as Faults with the Parameter's name as the Path.
func (a Action) Bind(given map[string]string) ([]Binding, error) {
	var faults Faults
	bindings := make([]Binding, 0, len(a.Params))
	declared := make(map[string]bool, len(a.Params))
	for _, p := range a.Params {
		declared[p.Name] = true
		v, ok := given[p.Name]
		if !ok && p.Default != nil {
			v, ok = *p.Default, true
		}
		if !ok {
			faults = append(faults, Fault{Path: p.Name, Message: "a value is required"})
			continue
		}
		rule, known := paramTypes[p.Type]
		if !known {
			faults = append(faults, Fault{Path: p.Name, Message: "the Parameter has an unknown type " + string(p.Type)})
			continue
		}
		if problem := rule.check(p, v); problem != "" {
			faults = append(faults, Fault{Path: p.Name, Message: "the value " + problem})
			continue
		}
		bindings = append(bindings, Binding{Name: p.Name, Value: v})
	}
	var unknown []string
	for name := range given {
		if !declared[name] {
			unknown = append(unknown, name)
		}
	}
	slices.Sort(unknown)
	for _, name := range unknown {
		faults = append(faults, Fault{Path: name, Message: "the Action " + a.ID + " has no Parameter with this name"})
	}
	if len(faults) > 0 {
		return nil, faults
	}
	return bindings, nil
}

// Environ returns the environment of a Run: base without any PULT_ variable,
// and then one PULT_ variable for each Binding. Removing the inherited PULT_
// variables keeps a name that no Parameter declares empty in the command.
func Environ(base []string, bindings []Binding) []string {
	env := make([]string, 0, len(base)+len(bindings))
	for _, kv := range base {
		if !strings.HasPrefix(kv, envPrefix) {
			env = append(env, kv)
		}
	}
	for _, b := range bindings {
		env = append(env, EnvName(b.Name)+"="+b.Value)
	}
	return env
}
