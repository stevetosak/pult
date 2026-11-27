// Package action is the only definition of Pult's Action file format
// (docs/adr/0003). The pult command on a Machine and the app's Core both use it.
//
// It parses and validates an Action file and gives the line of each fault
// ([Parse]), validates an Action that arrived as JSON ([Validate]), makes the
// canonical form of an Action and its pin ([Canonical], [Pin]), checks the
// values of a Run and maps them to the environment ([Action.Bind], [Environ]),
// and carries the values across the SSH hop ([RunArgs], [DecodeValues]).
package action

import "go.yaml.in/yaml/v4"

// FileVersion is the only Action file version this package reads.
const FileVersion = 1

// File is a parsed and valid Action file.
type File struct {
	Version int
	Actions []Action

	root *yaml.Node // the YAML, for the positions of Check; nil when not parsed
}

// Action returns the Action with the id.
func (f *File) Action(id string) (Action, bool) {
	for _, a := range f.Actions {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}

// Kind tells how an Action runs. The empty Kind is KindOutput.
type Kind string

const (
	// KindOutput runs with no terminal and no input; the app shows the output
	// and the exit code.
	KindOutput Kind = "output"
	// KindTerminal runs in the SSH session's terminal.
	KindTerminal Kind = "terminal"
)

// Action is a named command on a Machine. The JSON names are the YAML names.
type Action struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Group       string  `json:"group,omitempty"`
	Kind        Kind    `json:"kind,omitempty"`
	Confirm     bool    `json:"confirm,omitempty"`
	Params      []Param `json:"params,omitempty"`
	Run         string  `json:"run"`
}

// EffectiveKind returns the Kind, with the empty Kind read as KindOutput.
func (a Action) EffectiveKind() Kind {
	if a.Kind == "" {
		return KindOutput
	}
	return a.Kind
}

// ParamType is the type of a Parameter.
type ParamType string

const (
	TypeString ParamType = "string"
	TypeInt    ParamType = "int"
	TypeBool   ParamType = "bool"
	TypeChoice ParamType = "choice"
)

// Param is a typed value that the Operator gives when a Run starts. Which of
// Min, Max, Pattern and Options apply depends on the Type; a field that the
// Type does not take is a fault.
type Param struct {
	Name  string    `json:"name"`
	Title string    `json:"title"`
	Type  ParamType `json:"type"`
	// Default is the default value in its environment form (decimal for an
	// int, true or false for a bool). Nil means a value must be given.
	Default *string `json:"default,omitempty"`
	// Min and Max bound an int. For a string, Max is the most characters
	// (nil means 200).
	Min *int64 `json:"min,omitempty"`
	Max *int64 `json:"max,omitempty"`
	// Pattern is a Go RE2 expression that a string must match as a whole.
	Pattern string `json:"pattern,omitempty"`
	// Options are the values of a choice, in the order of the form.
	Options []string `json:"options,omitempty"`
}
