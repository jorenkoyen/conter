package manifest

import (
	"encoding/json"
	"io"
)

// Parse will read the incoming data and parse it to a [manifest.Project].
func Parse(r io.Reader) (*Project, error) {
	p := new(Project)
	err := json.NewDecoder(r).Decode(p)
	return p, err
}
