package types

type ChallengeType string

const (
	ChallengeTypeHTTP ChallengeType = "HTTP-01"
	ChallengeTypeDNS  ChallengeType = "DNS-01"
	ChallengeTypeTLS  ChallengeType = "TLS-ALPN-01"
	ChallengeTypeNone ChallengeType = "NONE"
)

type Ingress struct {
	Domain        string `json:"domain"`
	ContainerPort int    `json:"container_port"`

	TargetEndpoint string `json:"target_endpoint"`
	TargetService  string `json:"target_service"`
	TargetProject  string `json:"target_project"`

	ChallengeType ChallengeType `json:"challenge_type"`
}
