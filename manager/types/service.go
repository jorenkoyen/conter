package types

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/jorenkoyen/go-logger/log"
)

type Service struct {
	Name           string            `json:"name"`
	Hash           string            `json:"hash"`
	ContainerName  string            `json:"container_name"`
	ContainerImage string            `json:"container_image"`
	Source         Source            `json:"source"`
	Environment    map[string]string `json:"environment"`
	Quota          Quota             `json:"quota"`
	Ingress        Ingress           `json:"ingress"`
	Volumes        []Volume          `json:"volumes"`
}

type Source struct {
	Type string            `json:"type"`
	URI  string            `json:"uri"`
	Opts map[string]string `json:"opts"`
}

type Quota struct {
	MemoryLimit int64 `json:"memory_limit"`
}

type Volume struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// IsExposed will return true if the given service has listed ingress domains.
func (s *Service) IsExposed() bool {
	return len(s.Ingress.Domains) > 0
}

// CalculateHash will calculate the configuration hash for the specified service.
// This hash will be used to compare versions of the service.
func CalculateHash(s *Service) string {
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

	// include 'volumes'
	if len(s.Volumes) > 0 {
		if err := encoder.Encode(s.Volumes); err != nil {
			log.Panicf("Failed to hash volumes: %v", err)
		}
	}

	h := md5.New()
	h.Write(buf.Bytes())
	return fmt.Sprintf("%x", h.Sum(nil)) // hex string
}
