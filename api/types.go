package api

import (
	"fmt"
	"time"

	"github.com/jorenkoyen/conter/manager/types"
)

// StatusError is an error with an HTTP status code and message.
type StatusError struct {
	StatusCode   int
	Status       string
	ErrorMessage string `json:"error"`
}

func (e StatusError) Error() string {
	switch {
	case e.Status != "" && e.ErrorMessage != "":
		return fmt.Sprintf("%s: %s", e.Status, e.ErrorMessage)
	case e.Status != "":
		return e.Status
	case e.ErrorMessage != "":
		return e.ErrorMessage
	default:
		// this should not happen
		return "something went wrong, please see the Conter server logs for details"
	}
}

type Certificate struct {
	Domain    string              `json:"domain"`
	Challenge types.ChallengeType `json:"challenge"`
	PEM       string              `json:"pem,omitempty"`
	Meta      struct {
		Subject            string    `json:"subject"`
		Issuer             string    `json:"issuer"`
		Since              time.Time `json:"since"`
		Expiry             time.Time `json:"expiry"`
		SerialNumber       string    `json:"serial"`
		SignatureAlgorithm string    `json:"signature_algorithm"`
		PublicAlgorithm    string    `json:"public_algorithm"`
	} `json:"meta,omitempty"`
}
