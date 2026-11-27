package action

import (
	"encoding/json"
	"testing"
)

// The golden canonical forms and pins of goldenActions. They hold for all
// time: a change here is a breaking change and needs a new canonical form
// beside v1 (ADR 0003). The hashes were checked with sha256sum.
var golden = []struct {
	canonical string
	pin       string
}{
	{
		`{"id":"lock-screen","title":"Lock the screen","run":"loginctl lock-session"}`,
		"v1-1bdc36dce0cbde2808ebae37dc3df875013b1213bcc196e110df2fb176f5b766",
	},
	{
		`{"id":"unlock-vault","title":"Unlock the vault","description":"Asks for the master password — in the terminal.",` +
			`"group":"Vault","kind":"terminal","confirm":true,"params":[` +
			`{"name":"vault","title":"Vault","type":"choice","default":"work","options":["personal","work"]},` +
			`{"name":"hours","title":"Hours","type":"int","default":"2","min":0,"max":12},` +
			`{"name":"lid","title":"Lid","type":"bool","default":"false"},` +
			`{"name":"note","title":"Note","type":"string","max":40,"pattern":"[a-z <>&\"]*"}],` +
			`"run":"RBW_PROFILE=\"$PULT_VAULT\" rbw unlock && echo ok >/dev/null 2>&1 <&-\n"}`,
		"v1-9bc5e041a1b8fff7a27e55374fcb8e58bf8fac75add9ae84c7a6ed665270a7d7",
	},
}

func TestCanonicalGolden(t *testing.T) {
	actions := goldenActions()
	if len(actions) != len(golden) {
		t.Fatalf("%d golden Actions, %d golden forms", len(actions), len(golden))
	}
	for i, a := range actions {
		if err := Validate(a); err != nil {
			t.Fatalf("golden Action %s is not valid: %v", a.ID, err)
		}
		if got := string(Canonical(a)); got != golden[i].canonical {
			t.Errorf("Canonical(%s):\n  got  %s\n  want %s", a.ID, got, golden[i].canonical)
		}
		if got := Pin(a); got != golden[i].pin {
			t.Errorf("Pin(%s) = %s, want %s", a.ID, got, golden[i].pin)
		}
	}
}

func TestPinForm(t *testing.T) {
	pin := Pin(goldenActions()[0])
	if len(pin) != 67 || !safeArg(pin) {
		t.Errorf("pin %q: want 67 characters in the safe set", pin)
	}
}

// The same Action written in different ways has one pin: comments, spacing,
// quoting, an alias, and a field at its default written or left out.
func TestPinIgnoresTheWriting(t *testing.T) {
	plain := `version: 1
actions:
  - id: a
    title: A
    params:
      - {name: v, title: V, type: string}
      - {name: c, title: C, type: choice, options: [x, y]}
    run: echo "$PULT_V"
`
	written := `# a comment
version: 1
actions:
  - id: "a"        # quoted
    title: 'A'
    kind: output   # the default
    confirm: false # the default
    params: &p
      - name: v
        title: V
        type: string
        max: 200   # the default
      - name: c
        title: C
        type: choice
        options:
          - x
          - y
    run: >-
      echo "$PULT_V"
  - id: b
    title: B
    params: *p
    run: x
`
	want := pinOf(t, plain, "a")
	if got := pinOf(t, written, "a"); got != want {
		t.Errorf("pin of the written-out form = %s, want %s", got, want)
	}
}

// Every field counts: a change to any of them changes the pin.
func TestPinCoversEveryField(t *testing.T) {
	base := goldenActions()[1]
	basePin := Pin(base)
	changes := map[string]func(a *Action){
		"id":            func(a *Action) { a.ID = "unlock-vault2" },
		"title":         func(a *Action) { a.Title = "Unlock the Vault" },
		"description":   func(a *Action) { a.Description = "" },
		"group":         func(a *Action) { a.Group = "Vaults" },
		"kind":          func(a *Action) { a.Kind = KindOutput },
		"confirm":       func(a *Action) { a.Confirm = false },
		"run":           func(a *Action) { a.Run += " " },
		"param order":   func(a *Action) { a.Params[0], a.Params[1] = a.Params[1], a.Params[0] },
		"param removed": func(a *Action) { a.Params = a.Params[:3] },
		"param name":    func(a *Action) { a.Params[2].Name = "lid2" },
		"param title":   func(a *Action) { a.Params[2].Title = "Lid!" },
		"param default": func(a *Action) { a.Params[0].Default = ptr("personal") },
		"no default":    func(a *Action) { a.Params[0].Default = nil },
		"empty default": func(a *Action) { a.Params[3].Default = ptr("") },
		"min":           func(a *Action) { a.Params[1].Min = ptr[int64](1) },
		"max":           func(a *Action) { a.Params[1].Max = ptr[int64](11) },
		"string max":    func(a *Action) { a.Params[3].Max = nil },
		"pattern":       func(a *Action) { a.Params[3].Pattern = "" },
		"option order":  func(a *Action) { a.Params[0].Options = []string{"work", "personal"} },
		"option added":  func(a *Action) { a.Params[0].Options = []string{"personal", "work", "x"} },
	}
	for name, change := range changes {
		a := goldenActions()[1]
		change(&a)
		if Pin(a) == basePin {
			t.Errorf("%s: the pin did not change", name)
		}
	}
}

// The Core computes a pin from the Action it read as JSON from pult list
// --json; it must equal the pin that pult run computes from the file.
func TestPinSurvivesJSON(t *testing.T) {
	f, err := Parse([]byte(exampleFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range append(f.Actions, goldenActions()...) {
		data, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		var back Action
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatal(err)
		}
		if err := Validate(back); err != nil {
			t.Errorf("%s from JSON is not valid: %v", a.ID, err)
		}
		if Pin(back) != Pin(a) {
			t.Errorf("%s: pin from JSON %s, from the file %s", a.ID, Pin(back), Pin(a))
		}
	}
}

func pinOf(t *testing.T, src, id string) string {
	t.Helper()
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	a, ok := f.Action(id)
	if !ok {
		t.Fatalf("no Action %s", id)
	}
	return Pin(a)
}
