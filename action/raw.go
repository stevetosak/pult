package action

import (
	"fmt"
	"strconv"

	"go.yaml.in/yaml/v4"
)

// The raw types are the Action file as YAML gives it. Text fields decode as
// written (0755 stays "0755"). Numbers and booleans stay nodes, so that they
// are read strictly here: one spelling for each value.

type rawFile struct {
	Version yaml.Node   `yaml:"version"`
	Actions []rawAction `yaml:"actions"`
}

type rawAction struct {
	ID          string     `yaml:"id"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
	Group       string     `yaml:"group"`
	Kind        string     `yaml:"kind"`
	Confirm     yaml.Node  `yaml:"confirm"`
	Params      []rawParam `yaml:"params"`
	Run         string     `yaml:"run"`
}

type rawParam struct {
	Name    string    `yaml:"name"`
	Title   string    `yaml:"title"`
	Type    string    `yaml:"type"`
	Default yaml.Node `yaml:"default"`
	Min     yaml.Node `yaml:"min"`
	Max     yaml.Node `yaml:"max"`
	Pattern string    `yaml:"pattern"`
	Options []string  `yaml:"options"`
}

// decode reads the numbers and booleans strictly and returns the File.
func (raw rawFile) decode() (*File, Faults) {
	var d scalars
	f := &File{}
	if v := d.int(&raw.Version, path{"version"}); v != nil {
		f.Version = int(*v)
	}
	for i, ra := range raw.Actions {
		f.Actions = append(f.Actions, d.action(path{"actions", i}, ra))
	}
	return f, d.faults
}

func (d *scalars) action(at path, ra rawAction) Action {
	a := Action{
		ID:          ra.ID,
		Title:       ra.Title,
		Description: ra.Description,
		Group:       ra.Group,
		Kind:        Kind(ra.Kind),
		Confirm:     d.bool(&ra.Confirm, at.key("confirm")),
		Run:         ra.Run,
	}
	for i, rp := range ra.Params {
		pat := at.key("params").index(i)
		a.Params = append(a.Params, Param{
			Name:    rp.Name,
			Title:   rp.Title,
			Type:    ParamType(rp.Type),
			Default: d.defaultValue(&rp.Default, ParamType(rp.Type), pat.key("default")),
			Min:     d.int(&rp.Min, pat.key("min")),
			Max:     d.int(&rp.Max, pat.key("max")),
			Pattern: rp.Pattern,
			Options: rp.Options,
		})
	}
	return a
}

// scalars reads number and boolean nodes and collects their faults.
type scalars struct {
	faults Faults
}

func (d *scalars) fault(n *yaml.Node, at path, format string, args ...any) {
	d.faults = append(d.faults, Fault{
		Line: n.Line, Column: n.Column, Path: at.String(),
		Message: at[len(at)-1].(string) + " " + fmt.Sprintf(format, args...), at: at,
	})
}

// given returns the scalar of a field, or nil when the field is absent or
// null. A list or a mapping is a fault.
func (d *scalars) given(n *yaml.Node, at path) *yaml.Node {
	if n.Kind == 0 {
		return nil
	}
	n = resolve(n)
	if n.Kind != yaml.ScalarNode {
		d.fault(n, at, "must be one value, not %s", tagName(n.ShortTag()))
		return nil
	}
	if n.ShortTag() == "!!null" {
		return nil
	}
	return n
}

func (d *scalars) int(n *yaml.Node, at path) *int64 {
	s := d.given(n, at)
	if s == nil {
		return nil
	}
	// A plain number too large for YAML's int resolves as a float; the
	// decimal check below reports it as out of range.
	if tag := s.ShortTag(); tag != "!!int" && tag != "!!float" {
		d.fault(s, at, "%q is not a whole number", s.Value)
		return nil
	}
	v, problem := parseDecimal(s.Value)
	if problem != "" {
		d.fault(s, at, "%q %s", s.Value, problem)
		return nil
	}
	return &v
}

func (d *scalars) bool(n *yaml.Node, at path) bool {
	s := d.given(n, at)
	if s == nil {
		return false
	}
	if s.ShortTag() != "!!bool" || (s.Value != "true" && s.Value != "false") {
		d.fault(s, at, "%q is not true or false", s.Value)
		return false
	}
	return s.Value == "true"
}

// defaultValue returns a default in its environment form: decimal for an int,
// true or false for a bool, and the text as written for the other types.
func (d *scalars) defaultValue(n *yaml.Node, t ParamType, at path) *string {
	var v string
	switch t {
	case TypeInt:
		i := d.int(n, at)
		if i == nil {
			return nil
		}
		v = strconv.FormatInt(*i, 10)
	case TypeBool:
		if d.given(n, at) == nil {
			return nil
		}
		v = strconv.FormatBool(d.bool(n, at))
	default:
		s := d.given(n, at)
		if s == nil {
			return nil
		}
		v = s.Value
	}
	return &v
}
