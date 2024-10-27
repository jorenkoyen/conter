package manifest

// Project defines the internal structure of how a project should exist in the system.
// It contains all required information on how to manage and expose the services within the project.
type Project struct {
	Name     string    `json:"name"`
	Services []Service `json:"services"`
}

// Service contains the information on how a service should exist within the system.
type Service struct {
	Name   string `json:"name"`
	Source struct {
		Type string `json:"type"`
		URI  string `json:"uri"`
	} `json:"source"`
	Environment map[string]string `json:"environment"`
	Ingress     struct {
		Domain        string        `json:"domain"`
		ContainerPort int           `json:"container_port"`
		SslChallenge  ChallengeType `json:"ssl_challenge"`
	} `json:"ingress"`
}

// ChallengeType defines the ACME challenge to use when requesting a new SSL certificate.
type ChallengeType string

const (
	ChallengeHttp01 ChallengeType = "http01"
	ChallengeDns01  ChallengeType = "dns01"
)
