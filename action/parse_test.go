package action

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// exampleFile holds the first Actions with example values. The vault
// Parameter is written once and shared through an alias.
const exampleFile = `# Pult Actions of an example Machine
version: 1
actions:
  - id: unlock-vault
    title: Unlock the vault
    group: Vault
    kind: terminal
    params: &vault
      - name: vault
        title: Vault
        type: choice
        options: [personal, work]
        default: work
    run: RBW_PROFILE="$PULT_VAULT" exec rbw unlock

  - id: vault-status
    title: Vault status
    group: Vault
    params: *vault
    run: RBW_PROFILE="$PULT_VAULT" rbw unlocked && echo unlocked

  - id: keep-awake
    title: Keep awake
    description: Holds a sleep inhibitor for some hours.
    group: Session
    params:
      - name: hours
        title: Hours
        type: int
        min: 1
        max: 12
        default: 2
      - name: lid
        title: Also with the lid closed
        type: bool
        default: false
    run: |
      what=sleep:idle
      if [ "$PULT_LID" = true ]; then what="$what:handle-lid-switch"; fi
      setsid -f systemd-inhibit --what="$what" --who=pult --why="Pult keep-awake" \
        sleep "${PULT_HOURS}h" >/dev/null 2>&1

  - id: lock-screen
    title: Lock the screen
    confirm: true
    run: loginctl lock-session

  - id: open-file
    title: Open a file
    params:
      - name: mode
        title: Mode
        type: choice
        options: [no, yes, 0755, 1.10]
      - name: note
        title: Note
        type: string
        max: 40
        pattern: '[a-z ]*'
    run: echo "$PULT_MODE $PULT_NOTE"
`

func TestParseExampleFile(t *testing.T) {
	f, err := Parse([]byte(exampleFile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ids := make([]string, len(f.Actions))
	for i, a := range f.Actions {
		ids[i] = a.ID
	}
	if want := []string{"unlock-vault", "vault-status", "keep-awake", "lock-screen", "open-file"}; !slices.Equal(ids, want) {
		t.Fatalf("ids = %v, want %v (the order of the file)", ids, want)
	}

	unlock, _ := f.Action("unlock-vault")
	status, _ := f.Action("vault-status")
	if unlock.Kind != KindTerminal || status.EffectiveKind() != KindOutput {
		t.Errorf("kinds = %q, %q", unlock.Kind, status.EffectiveKind())
	}
	if got := status.Params[0]; got.Name != "vault" || *got.Default != "work" || !slices.Equal(got.Options, []string{"personal", "work"}) {
		t.Errorf("the alias did not give vault-status the vault Parameter: %+v", got)
	}

	awake, _ := f.Action("keep-awake")
	hours, lid := awake.Params[0], awake.Params[1]
	if *hours.Default != "2" || *hours.Min != 1 || *hours.Max != 12 {
		t.Errorf("hours = default %q, min %d, max %d", *hours.Default, *hours.Min, *hours.Max)
	}
	if *lid.Default != "false" {
		t.Errorf("lid default = %q, want false", *lid.Default)
	}
	if !strings.HasSuffix(awake.Run, ">/dev/null 2>&1\n") {
		t.Errorf("run lost its text: %q", awake.Run)
	}

	lock, _ := f.Action("lock-screen")
	if !lock.Confirm {
		t.Error("lock-screen: confirm = false, want true")
	}

	open, _ := f.Action("open-file")
	if got, want := open.Params[0].Options, []string{"no", "yes", "0755", "1.10"}; !slices.Equal(got, want) {
		t.Errorf("options = %q, want %q as written", got, want)
	}

	if _, ok := f.Action("missing"); ok {
		t.Error(`Action("missing") was found`)
	}
}

// file returns a valid file with one Action whose lines below "- id: a" are
// the given lines, indented by four spaces.
func file(lines ...string) string {
	var b strings.Builder
	b.WriteString("version: 1\nactions:\n  - id: a\n")
	for _, l := range lines {
		b.WriteString("    " + l + "\n")
	}
	return b.String()
}

func TestParseFaults(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		// The YAML itself.
		{"empty file", "", []string{"1:1: the file is empty; it needs version: 1 and actions"}},
		{"comments only", "# nothing\n", []string{"1:1: the file is empty; it needs version: 1 and actions"}},
		{"invalid UTF-8", "version: 1\nactions:\n  - id: \xff\n", []string{"3:9: the file is not valid UTF-8"}},
		{"two documents", "version: 1\n---\nversion: 2\n", []string{"2:1: the file holds more than one YAML document"}},
		{"syntax", "version: 1\nactions:\n  - id: a\n   run: x\n", []string{"4:4: did not find expected '-' indicator"}},
		{"merge key", "version: 1\nactions:\n  - &a\n    id: a\n    title: A\n    run: x\n  - <<: *a\n    id: b\n",
			[]string{"7:5: merge keys (<<) are not supported; use an alias of the whole value"}},

		// The types of the fields.
		{"root is a list", "- a\n", []string{"1:1: expected a mapping, found a list"}},
		{"unknown field", file("title: A", "run: x", "params:", "  - name: v", "    title: V", "    type: string", "    maxx: 3"),
			[]string{`10:9: unknown field "maxx"`}},
		{"duplicate key", file("title: A", "run: x", "run: y"), []string{`6:5: mapping key "run" already defined at line 5`}},
		{"list for text", file("title: [A]", "run: x"), []string{"4:12: expected text, found a list"}},
		{"version as text", "version: one\nactions: []\n", []string{`1:10: version "one" is not a whole number`}},
		{"confirm yes", file("title: A", "confirm: yes", "run: x"), []string{`5:14: confirm "yes" is not true or false`}},
		{"confirm True", file("title: A", "confirm: True", "run: x"), []string{`5:14: confirm "True" is not true or false`}},
		{"confirm list", file("title: A", "confirm: [true]", "run: x"), []string{"5:14: confirm must be one value, not a list"}},
		{"max in hex", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1, max: 0x0C}"),
			[]string{`7:53: max "0x0C" is not a whole number in decimal`}},
		{"min quoted", file("title: A", "run: x", "params:", `  - {name: v, title: V, type: int, min: "1", max: 2}`),
			[]string{`7:45: min "1" is not a whole number`}},
		{"min as float", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1.0, max: 2}"),
			[]string{`7:45: min "1.0" is not a whole number in decimal`}},
		{"int default leading zero", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1, max: 900, default: 0755}"),
			[]string{`7:67: default "0755" is not a whole number in decimal`}},
		{"int too large", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1, max: 99999999999999999999}"),
			[]string{`7:53: max "99999999999999999999" is out of the range of a 64-bit whole number`}},
		{"default list", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: string, default: [a]}"),
			[]string{"7:52: default must be one value, not a list"}},
		{"type faults in line order", file("confirm: on", "title: [A]", "run: x"),
			[]string{`4:14: confirm "on" is not true or false`, "5:12: expected text, found a list"}},

		// The rules of the format.
		{"no version", "actions: []\n", []string{"1:1: version is required; this pult reads version 1"}},
		{"version 2", "version: 2\nactions: []\n", []string{"1:10: version 2 is not supported; this pult reads version 1"}},
		{"id missing", "version: 1\nactions:\n  - title: A\n    run: x\n", []string{"3:5: id is required"}},
		{"id form", "version: 1\nactions:\n  - id: Keep\n    title: A\n    run: x\n",
			[]string{`3:9: id "Keep" must match ^[a-z][a-z0-9-]{0,47}$`}},
		{"id twice", "version: 1\nactions:\n  - id: a\n    title: A\n    run: x\n  - id: a\n    title: B\n    run: y\n",
			[]string{`6:9: id "a" is already used at line 3`}},
		{"title missing", file("run: x"), []string{"3:5: title is required"}},
		{"title long", file("title: "+strings.Repeat("é", 61), "run: x"), []string{"4:12: title is longer than 60 characters"}},
		{"title tab", file(`title: "A\tB"`, "run: x"), []string{"4:12: title holds a control character"}},
		{"title bidi", file("title: \"A‮B\"", "run: x"), []string{"4:12: title holds a control character"}},
		{"title newline", file("title: |", "  two", "  lines", "run: x"), []string{"4:12: title holds a control character"}},
		{"description newline", file("title: A", "description: |", "  two", "  lines", "run: x"), nil},
		{"description long", file("title: A", "description: "+strings.Repeat("a", 201), "run: x"),
			[]string{"5:18: description is longer than 200 characters"}},
		{"group long", file("title: A", "group: "+strings.Repeat("g", 31), "run: x"), []string{"5:12: group is longer than 30 characters"}},
		{"kind", file("title: A", "kind: background", "run: x"), []string{`5:11: kind "background" is not output or terminal`}},
		{"run missing", file("title: A"), []string{"3:5: run is required"}},
		{"run blank", file("title: A", `run: "  "`), []string{"5:10: run is required"}},
		{"run long", file("title: A", "run: "+strings.Repeat("x", 4097)), []string{"5:10: run is 4097 bytes, more than 4096"}},
		{"run NUL", file("title: A", `run: "a\0b"`), []string{"5:10: run holds a NUL character"}},
		{"nine params", file(append([]string{"title: A", "run: x", "params:"},
			repeatLines(9, "  - {name: p%d, title: P, type: bool}")...)...),
			[]string{"7:7: an Action has at most 8 Parameters, this one has 9"}},
		{"param name form", file("title: A", "run: x", "params:", "  - {name: Hours, title: H, type: bool}"),
			[]string{`7:16: name "Hours" must match ^[a-z][a-z0-9_]{0,31}$`}},
		{"param name twice", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: bool}", "  - {name: v, title: W, type: bool}"),
			[]string{`8:16: Parameter name "v" is already used at line 7`}},
		{"param title missing", file("title: A", "run: x", "params:", "  - {name: v, type: bool}"), []string{"7:9: title is required"}},
		{"type missing", file("title: A", "run: x", "params:", "  - {name: v, title: V}"),
			[]string{"7:9: type is required: one of bool, choice, int, string"}},
		{"type unknown", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: float}"),
			[]string{`7:35: type "float" is not one of bool, choice, int, string`}},
		{"int without limits", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int}"),
			[]string{"7:9: min is required for an int Parameter", "7:9: max is required for an int Parameter"}},
		{"int min above max", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 5, max: 1}"),
			[]string{"7:45: min 5 is above max 1"}},
		{"int default above max", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1, max: 12, default: 13}"),
			[]string{"7:66: default is above the maximum 12"}},
		{"int with options", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: int, min: 1, max: 2, options: [a]}"),
			[]string{"7:65: options is not a field of a int Parameter"}},
		{"bool with max", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: bool, max: 1}"),
			[]string{"7:46: max is not a field of a bool Parameter"}},
		{"bool default text", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: bool, default: yes}"),
			[]string{`7:50: default "yes" is not true or false`}},
		{"string max", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: string, max: 2000}"),
			[]string{"7:48: max must be from 1 to 1024 for a string, not 2000"}},
		{"string pattern", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: string, pattern: '(a'}"),
			[]string{"7:52: pattern is not a valid RE2 expression: error parsing regexp: missing closing ): `(a`"}},
		{"string default pattern", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: string, pattern: '[a-z]+', default: A1}"),
			[]string{`7:71: default does not match the pattern "[a-z]+"`}},
		{"string default long", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: string, max: 2, default: abc}"),
			[]string{"7:60: default is longer than 2 characters"}},
		{"choice without options", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: choice}"),
			[]string{"7:9: options is required for a choice Parameter, with at least one option"}},
		{"choice option twice", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: choice, options: [a, b, a]}"),
			[]string{`7:59: option "a" is already listed at line 7`}},
		{"choice empty option", file("title: A", "run: x", "params:", `  - {name: v, title: V, type: choice, options: [a, ""]}`),
			[]string{"7:56: option is required"}},
		{"choice default", file("title: A", "run: x", "params:", "  - {name: v, title: V, type: choice, options: [a], default: b}"),
			[]string{"7:66: default is not one of the options"}},
		{"rule faults in line order", "version: 1\nactions:\n  - id: B\n    run: x\n  - id: c\n    title: C\n    kind: k\n",
			[]string{"3:5: title is required", `3:9: id "B" must match ^[a-z][a-z0-9-]{0,47}$`,
				"5:5: run is required", `7:11: kind "k" is not output or terminal`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.src))
			if got := faultStrings(t, err); !slices.Equal(got, tt.want) {
				t.Errorf("faults:\n  got  %q\n  want %q", got, tt.want)
			}
		})
	}
}

func TestParseRefusesLargeFile(t *testing.T) {
	src := "version: 1\nactions: []\n#" + strings.Repeat("x", maxFileSize)
	_, err := Parse([]byte(src))
	if got := faultStrings(t, err); len(got) != 1 || !strings.Contains(got[0], "larger than") {
		t.Errorf("faults = %q", got)
	}
}

func faultStrings(t *testing.T, err error) []string {
	t.Helper()
	if err == nil {
		return nil
	}
	var faults Faults
	if !errors.As(err, &faults) {
		t.Fatalf("error %T is not Faults: %v", err, err)
	}
	out := make([]string, len(faults))
	for i, f := range faults {
		out[i] = f.String()
	}
	return out
}

func repeatLines(n int, format string) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = strings.ReplaceAll(format, "%d", string(rune('a'+i)))
	}
	return lines
}
