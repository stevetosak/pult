package action

func ptr[T any](v T) *T { return &v }

// goldenActions are fixed Actions whose canonical forms and pins must never
// change (canonical_test.go).
func goldenActions() []Action {
	return []Action{
		{ID: "lock-screen", Title: "Lock the screen", Run: "loginctl lock-session"},
		{
			ID:          "unlock-vault",
			Title:       "Unlock the vault",
			Description: "Asks for the master password — in the terminal.",
			Group:       "Vault",
			Kind:        KindTerminal,
			Confirm:     true,
			Params: []Param{
				{Name: "vault", Title: "Vault", Type: TypeChoice, Options: []string{"personal", "work"}, Default: ptr("work")},
				{Name: "hours", Title: "Hours", Type: TypeInt, Min: ptr[int64](0), Max: ptr[int64](12), Default: ptr("2")},
				{Name: "lid", Title: "Lid", Type: TypeBool, Default: ptr("false")},
				{Name: "note", Title: "Note", Type: TypeString, Max: ptr[int64](40), Pattern: `[a-z <>&"]*`},
			},
			Run: "RBW_PROFILE=\"$PULT_VAULT\" rbw unlock && echo ok >/dev/null 2>&1 <&-\n",
		},
	}
}
