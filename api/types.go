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
	ID        string              `json:"id"`
	Challenge types.ChallengeType `json:"challenge"`
	Domains   []string            `json:"domains"`
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

type ProjectSummary struct {
	Name     string   `json:"name"`
	Running  bool     `json:"running"`
	Services []string `json:"services"`
}

type Project struct {
	Name     string    `json:"project"`
	Services []Service `json:"services"`
}

type Service struct {
	Name    string   `json:"name"`
	Hash    string   `json:"hash"`
	Status  string   `json:"status"`
	Volumes []string `json:"volumes"`
	Ingress struct {
		Domains          []string            `json:"domains"`
		InternalEndpoint string              `json:"internal"`
		ChallengeType    types.ChallengeType `json:"challenge"`
	} `json:"ingress,omitempty"`
}

type ProjectApplyCommand struct {
	ProjectName string `json:"project_name"`
	Services    []struct {
		Name          string              `json:"name"`
		Source        types.Source        `json:"source"`
		Environment   map[string]string   `json:"environment"`
		IngressDomain []string            `json:"ingress_domains"`
		ContainerPort int                 `json:"container_port"`
		ChallengeType types.ChallengeType `json:"challenge_type"`
		Quota         types.Quota         `json:"quota"`
		Volumes       []types.Volume      `json:"volumes"`
	} `json:"services"`
}

type Task string

const (
	TaskCertificateBatch Task = "batch_certificates"
)
