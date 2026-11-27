package action

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Fault is one problem in an Action file, an Action or the values of a Run.
type Fault struct {
	// Line and Column give the position in the Action file, from 1. Both are
	// 0 when the fault has no position in a file.
	Line   int
	Column int
	// Path names the field, for example actions[1].params[0].max, or the
	// Parameter of a value.
	Path    string
	Message string

	at path
}

// String returns the fault as "line:column: message" when it has a position,
// and as "path: message" when it does not.
func (f Fault) String() string {
	switch {
	case f.Line > 0:
		return fmt.Sprintf("%d:%d: %s", f.Line, f.Column, f.Message)
	case f.Path != "":
		return f.Path + ": " + f.Message
	default:
		return f.Message
	}
}

// Faults is the error that Parse, Validate and Bind return: every fault that
// was found. Faults from Parse are in line order.
type Faults []Fault

func (fs Faults) Error() string {
	lines := make([]string, len(fs))
	for i, f := range fs {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

func (fs Faults) sortByPosition() {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Line != fs[j].Line {
			return fs[i].Line < fs[j].Line
		}
		return fs[i].Column < fs[j].Column
	})
}

func (fs Faults) err() error {
	if len(fs) == 0 {
		return nil
	}
	return fs
}

// path names a field: map keys as strings, list positions as ints.
type path []any

func (p path) key(k string) path { return append(p[:len(p):len(p)], k) }
func (p path) index(i int) path  { return append(p[:len(p):len(p)], i) }

func (p path) String() string {
	var b strings.Builder
	for _, el := range p {
		switch v := el.(type) {
		case string:
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(v)
		case int:
			b.WriteString("[" + strconv.Itoa(v) + "]")
		}
	}
	return b.String()
}

// report collects the faults of one validation. where names the place of an
// earlier field in a message ("line 5" for a file, the path for JSON).
type report struct {
	faults Faults
	where  func(path) string
}

func newReport() *report {
	return &report{where: func(p path) string { return p.String() }}
}

func (r *report) add(at path, format string, args ...any) {
	r.faults = append(r.faults, Fault{Path: at.String(), Message: fmt.Sprintf(format, args...), at: at})
}
