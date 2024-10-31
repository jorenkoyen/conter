package manifest

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/jorenkoyen/go-logger/log"
)

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
	Quota struct {
		MemoryLimit int64 `json:"memory_limit"`
	} `json:"quota"`
}

// CalculateConfigurationHash will return a hash that can be used to compare configurations.
// If the hash calculation fails for whatever reason this function will panic.
func (s *Service) CalculateConfigurationHash() string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	// include 'source'
	if err := encoder.Encode(s.Source); err != nil {
		log.Panicf("Failed to hash source: %v", err)
	}

	// include 'environment'
	if err := encoder.Encode(s.Environment); err != nil {
		log.Panicf("Failed to hash environment: %v", err)
	}

	// include 'container_port'
	if err := encoder.Encode(s.Ingress.ContainerPort); err != nil {
		log.Panicf("Failed to hash container port: %v", err)
	}

	// include 'quota'
	if err := encoder.Encode(s.Quota); err != nil {
		log.Panicf("Failed to hash quota: %v", err)
	}

	// hash binary content
	h := md5.New()
	h.Write(buf.Bytes())
	return fmt.Sprintf("%x", h.Sum(nil)) // output as hex string
}

// HasIngress will return true if the service has all required fields for forwarding ingress traffic.
func (s *Service) HasIngress() bool {
	return s.Ingress.Domain != "" && s.Ingress.ContainerPort > 0
}

// ChallengeType defines the ACME challenge to use when requesting a new SSL certificate.
type ChallengeType string

const (
	ChallengeHttp01 ChallengeType = "http01"
	ChallengeDns01  ChallengeType = "dns01"
)
