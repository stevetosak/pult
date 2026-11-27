package action

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// maxListing is the largest `pult list --json` output that DecodeListing reads.
const maxListing = 4 << 20

// Listing is what `pult list --json` prints and the Core reads: the version of
// the Action file format and the Actions in the order of the file. It holds no
// pins: the Core computes each pin from the Action it shows.
type Listing struct {
	Version int      `json:"version"`
	Actions []Action `json:"actions"`
}

// NewListing returns the Listing of a parsed File.
func NewListing(f *File) Listing {
	actions := f.Actions
	if actions == nil {
		actions = []Action{}
	}
	return Listing{Version: f.Version, Actions: actions}
}

// DecodeListing reads the output of `pult list --json`. It refuses a field it
// does not know, because a newer pult can add a field that its pins cover:
// an older Core would then compute other pins, and every Run would be
// refused. It checks each Action with Validate and refuses an id used twice.
func DecodeListing(data []byte) (Listing, error) {
	if len(data) > maxListing {
		return Listing{}, fmt.Errorf("the listing is %d bytes, more than %d", len(data), maxListing)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var l Listing
	if err := dec.Decode(&l); err != nil {
		return Listing{}, fmt.Errorf("the listing is not valid: %w", err)
	}
	if dec.More() {
		return Listing{}, fmt.Errorf("the listing has more than one JSON value")
	}
	if l.Version != FileVersion {
		return Listing{}, fmt.Errorf("the listing has version %d; this Pult reads version %d", l.Version, FileVersion)
	}
	r := newReport()
	validateActions(r, l.Actions)
	if err := r.faults.err(); err != nil {
		return Listing{}, err
	}
	return l, nil
}
