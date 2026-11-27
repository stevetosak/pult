package action

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The limits of ADR 0003, and the ones the code adds (ADR 0003, amendment
// 2026-11-27).
const (
	maxTitle       = 60
	maxDescription = 200
	maxGroup       = 30
	maxRunBytes    = 4096
	maxParams      = 8
)

var (
	idPattern        = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
	paramNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
)

// Validate checks one Action against the rules of the Action file format. The
// Core uses it for an Action that it read from `pult list --json`; the faults
// name their fields by path, as params[0].max.
func Validate(a Action) error {
	r := newReport()
	validateAction(r, nil, a)
	return r.faults.err()
}

func validateFile(r *report, f *File) {
	if f.Version != FileVersion {
		if f.Version == 0 {
			r.add(path{"version"}, "version is required; this pult reads version %d", FileVersion)
		} else {
			r.add(path{"version"}, "version %d is not supported; this pult reads version %d", f.Version, FileVersion)
		}
	}
	firstID := map[string]path{}
	for i, a := range f.Actions {
		at := path{"actions", i}
		validateAction(r, at, a)
		if a.ID == "" {
			continue
		}
		if first, used := firstID[a.ID]; used {
			r.add(at.key("id"), "id %q is already used at %s", a.ID, r.where(first.key("id")))
			continue
		}
		firstID[a.ID] = at
	}
}

func validateAction(r *report, at path, a Action) {
	switch {
	case a.ID == "":
		r.add(at.key("id"), "id is required")
	case !idPattern.MatchString(a.ID):
		r.add(at.key("id"), "id %q must match %s", a.ID, idPattern)
	}
	checkText(r, at.key("title"), "title", a.Title, textRule{required: true, max: maxTitle})
	checkText(r, at.key("description"), "description", a.Description, textRule{max: maxDescription, multiLine: true})
	checkText(r, at.key("group"), "group", a.Group, textRule{max: maxGroup})
	switch a.Kind {
	case "", KindOutput, KindTerminal:
	default:
		r.add(at.key("kind"), "kind %q is not %s or %s", a.Kind, KindOutput, KindTerminal)
	}
	checkRun(r, at.key("run"), a.Run)

	if len(a.Params) > maxParams {
		r.add(at.key("params"), "an Action has at most %d Parameters, this one has %d", maxParams, len(a.Params))
	}
	firstName := map[string]path{}
	for i, p := range a.Params {
		pat := at.key("params").index(i)
		validateParam(r, pat, p)
		if p.Name == "" {
			continue
		}
		if first, used := firstName[p.Name]; used {
			r.add(pat.key("name"), "Parameter name %q is already used at %s", p.Name, r.where(first.key("name")))
			continue
		}
		firstName[p.Name] = pat
	}
}

func checkRun(r *report, at path, run string) {
	switch {
	case strings.TrimSpace(run) == "":
		r.add(at, "run is required")
	case len(run) > maxRunBytes:
		r.add(at, "run is %d bytes, more than %d", len(run), maxRunBytes)
	case !utf8.ValidString(run):
		r.add(at, "run is not valid UTF-8")
	case strings.ContainsRune(run, 0):
		r.add(at, "run holds a NUL character")
	}
}

func validateParam(r *report, at path, p Param) {
	switch {
	case p.Name == "":
		r.add(at.key("name"), "name is required")
	case !paramNamePattern.MatchString(p.Name):
		r.add(at.key("name"), "name %q must match %s", p.Name, paramNamePattern)
	}
	checkText(r, at.key("title"), "title", p.Title, textRule{required: true, max: maxTitle})

	rule, known := paramTypes[p.Type]
	if !known {
		if p.Type == "" {
			r.add(at.key("type"), "type is required: one of %s", typeNames())
		} else {
			r.add(at.key("type"), "type %q is not one of %s", p.Type, typeNames())
		}
		return
	}
	takes := map[string]bool{}
	for _, f := range rule.fields() {
		takes[f] = true
	}
	for _, f := range limitFields(p) {
		if !takes[f] {
			r.add(at.key(f), "%s is not a field of a %s Parameter", f, p.Type)
		}
	}
	rule.validate(r, at, p)
	if p.Default != nil {
		if problem := rule.check(p, *p.Default); problem != "" {
			r.add(at.key("default"), "default %s", problem)
		}
	}
}

// limitFields lists the limit fields that a Parameter gives.
func limitFields(p Param) []string {
	var given []string
	if p.Min != nil {
		given = append(given, "min")
	}
	if p.Max != nil {
		given = append(given, "max")
	}
	if p.Pattern != "" {
		given = append(given, "pattern")
	}
	if p.Options != nil {
		given = append(given, "options")
	}
	return given
}

// textRule is how a text field is checked.
type textRule struct {
	required  bool
	max       int  // in characters
	multiLine bool // newlines are allowed
}

func checkText(r *report, at path, field, s string, rule textRule) {
	if problem := textProblem(s, rule); problem != "" {
		r.add(at, "%s %s", field, problem)
	}
}

// textProblem returns what is wrong with a text, or "".
func textProblem(s string, rule textRule) string {
	switch {
	case !utf8.ValidString(s):
		return "is not valid UTF-8"
	case rule.required && strings.TrimSpace(s) == "":
		return "is required"
	case utf8.RuneCountInString(s) > rule.max:
		return "is longer than " + strconv.Itoa(rule.max) + " characters"
	}
	for _, c := range s {
		if c == '\n' && rule.multiLine {
			continue
		}
		if isControl(c) {
			return "holds a control character"
		}
	}
	return ""
}

// isControl reports a character that no text or value may hold: a control
// character (newline, ESC, NUL, DEL and the C1 set) or a bidirectional
// control, which can make a title read as something else on the screen.
func isControl(c rune) bool {
	return unicode.IsControl(c) || unicode.Is(unicode.Bidi_Control, c)
}
