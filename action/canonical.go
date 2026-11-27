package action

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// canonicalVersion names the canonical form. A change that alters a golden
// hash needs a new form beside v1, never an edit of v1 (ADR 0003).
const canonicalVersion = "v1"

// The v1 canonical form: one fixed struct per level, the fields in this order,
// only strings, integers and booleans, and each field at its default value
// left out. A field that a later version adds must be left out at its default,
// so that the pins of older Actions stay as they were.
type canonicalAction struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description,omitempty"`
	Group       string           `json:"group,omitempty"`
	Kind        Kind             `json:"kind,omitempty"`
	Confirm     bool             `json:"confirm,omitempty"`
	Params      []canonicalParam `json:"params,omitempty"`
	Run         string           `json:"run"`
}

type canonicalParam struct {
	Name    string    `json:"name"`
	Title   string    `json:"title"`
	Type    ParamType `json:"type"`
	Default *string   `json:"default,omitempty"`
	Min     *int64    `json:"min,omitempty"`
	Max     *int64    `json:"max,omitempty"`
	Pattern string    `json:"pattern,omitempty"`
	Options []string  `json:"options,omitempty"`
}

// Canonical returns the v1 canonical form of a valid Action: JSON of its
// fields in a fixed order, with no spaces, and with each field at its default
// value left out (kind output, confirm false, a string's max 200).
func Canonical(a Action) []byte {
	c := canonicalAction{
		ID:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		Group:       a.Group,
		Confirm:     a.Confirm,
		Run:         a.Run,
	}
	if a.EffectiveKind() != KindOutput {
		c.Kind = a.Kind
	}
	for _, p := range a.Params {
		if rule, known := paramTypes[p.Type]; known {
			p = rule.canonical(p)
		}
		c.Params = append(c.Params, canonicalParam(p))
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		// Only strings, integers and booleans: encoding cannot fail.
		panic("action: canonical form: " + err.Error())
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

// Pin returns the Action pin of a valid Action: "v1-" and the SHA-256 of its
// canonical form in lowercase hex.
func Pin(a Action) string {
	sum := sha256.Sum256(Canonical(a))
	return canonicalVersion + "-" + hex.EncodeToString(sum[:])
}
