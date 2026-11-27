package action

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v4"
)

// maxFileSize is the largest Action file that Parse reads.
const maxFileSize = 1 << 20

// Parse reads an Action file. When the file has faults, it returns Faults with
// the line and column of each, in line order.
//
// Parse works in stages, and a stage with faults stops the ones after it: the
// YAML itself, then the types of the fields, then the rules of the format. All
// faults of one stage are reported together.
func Parse(src []byte) (*File, error) {
	root, faults := loadDocument(src)
	if faults != nil {
		return nil, faults
	}
	// A field of the wrong type stays at its zero value and the library goes
	// on, so the strict numbers and booleans are read even after a fault.
	var raw rawFile
	loadErr := root.Load(&raw, yaml.WithKnownFields())
	file, faults := raw.decode()
	if loadErr != nil {
		faults = append(loadFaults(loadErr), faults...)
	}
	if faults != nil {
		faults.sortByPosition()
		return nil, faults
	}
	r := newReport()
	r.where = func(p path) string {
		n := locate(root, p)
		return "line " + strconv.Itoa(n.Line)
	}
	validateFile(r, file)
	if len(r.faults) > 0 {
		for i := range r.faults {
			n := locate(root, r.faults[i].at)
			r.faults[i].Line, r.faults[i].Column = n.Line, n.Column
		}
		r.faults.sortByPosition()
		return nil, r.faults
	}
	return file, nil
}

// loadDocument reads the YAML of an Action file into its one document node.
func loadDocument(src []byte) (*yaml.Node, Faults) {
	if len(src) > maxFileSize {
		return nil, Faults{{Line: 1, Column: 1, Message: fmt.Sprintf("the file is larger than %d bytes", maxFileSize)}}
	}
	if f, bad := invalidUTF8(src); bad {
		return nil, Faults{f}
	}
	var docs []yaml.Node
	if err := yaml.Load(src, &docs, yaml.WithAllDocuments()); err != nil {
		return nil, loadFaults(err)
	}
	switch {
	case len(docs) == 0:
		return nil, Faults{{Line: 1, Column: 1, Message: "the file is empty; it needs version: 1 and actions"}}
	case len(docs) > 1:
		return nil, Faults{{Line: docs[1].Line, Column: docs[1].Column, Message: "the file holds more than one YAML document"}}
	}
	if faults := mergeKeyFaults(&docs[0]); faults != nil {
		return nil, faults
	}
	return &docs[0], nil
}

// invalidUTF8 finds the first byte that is not UTF-8. The YAML reader refuses
// it too, but without a position.
func invalidUTF8(src []byte) (Fault, bool) {
	line, col := 1, 1
	for i := 0; i < len(src); {
		c, size := utf8.DecodeRune(src[i:])
		if c == utf8.RuneError && size == 1 {
			return Fault{Line: line, Column: col, Message: "the file is not valid UTF-8"}, true
		}
		if c == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
		i += size
	}
	return Fault{}, false
}

// mergeKeyFaults refuses YAML 1.1 merge keys (<<). Anchors and aliases of
// whole values stay allowed.
func mergeKeyFaults(n *yaml.Node) Faults {
	var faults Faults
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				if k := n.Content[i]; k.ShortTag() == "!!merge" {
					faults = append(faults, Fault{Line: k.Line, Column: k.Column,
						Message: "merge keys (<<) are not supported; use an alias of the whole value"})
				}
			}
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(n)
	return faults
}

// loadFaults turns an error of the YAML library into Faults.
func loadFaults(err error) Faults {
	var many *yaml.LoadErrors
	if errors.As(err, &many) {
		faults := make(Faults, 0, len(many.Errors))
		for _, e := range many.Errors {
			faults = append(faults, loadFault(e))
		}
		faults.sortByPosition()
		return faults
	}
	var one *yaml.LoadError
	if errors.As(err, &one) {
		return Faults{loadFault(one)}
	}
	return Faults{{Message: err.Error()}}
}

var (
	unknownFieldMessage = regexp.MustCompile("^field (\\S+) not found in type \\S+$")
	constructMessage    = regexp.MustCompile("^cannot construct (!!\\w+)(?: `(.*)`)? into (\\S+)$")
)

// loadFault writes a fault of the YAML library in the words of the format,
// without Go type names.
func loadFault(e *yaml.LoadError) Fault {
	msg := e.Message
	if m := unknownFieldMessage.FindStringSubmatch(msg); m != nil {
		msg = fmt.Sprintf("unknown field %q", m[1])
	} else if m := constructMessage.FindStringSubmatch(msg); m != nil {
		found := tagName(m[1])
		if m[2] != "" {
			found += " " + strconv.Quote(m[2])
		}
		msg = fmt.Sprintf("expected %s, found %s", goTypeName(m[3]), found)
	}
	return Fault{Line: e.Mark.Line, Column: e.Mark.Column, Message: msg}
}

func tagName(tag string) string {
	switch tag {
	case "!!seq":
		return "a list"
	case "!!map":
		return "a mapping"
	case "!!str":
		return "text"
	case "!!int", "!!float":
		return "a number"
	case "!!bool":
		return "a boolean"
	case "!!null":
		return "nothing"
	default:
		return tag
	}
}

func goTypeName(t string) string {
	switch {
	case t == "string":
		return "text"
	case strings.HasPrefix(t, "[]"):
		return "a list"
	case strings.Contains(t, "."):
		return "a mapping"
	default:
		return t
	}
}

// locate returns the node of a field, or the nearest node above it that the
// file has, such as the Action that lacks a required field.
func locate(root *yaml.Node, at path) *yaml.Node {
	n := root
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	for _, el := range at {
		next := child(resolve(n), el)
		if next == nil {
			break
		}
		n = next
	}
	return n
}

func child(n *yaml.Node, el any) *yaml.Node {
	switch k := el.(type) {
	case string:
		if n.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == k {
				return n.Content[i+1]
			}
		}
	case int:
		if n.Kind == yaml.SequenceNode && k < len(n.Content) {
			return n.Content[k]
		}
	}
	return nil
}

// resolve follows an alias to the node it names.
func resolve(n *yaml.Node) *yaml.Node {
	for n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	return n
}
