package docker

import (
	"fmt"
	"github.com/jorenkoyen/conter/manifest"
	"net"
)

const (
	LabelManagedBy = "conter.managed"
	LabelHash      = "conter.hash"
	LabelName      = "conter.name"

	ApplicationName = "conter"

	PortStartRange = 30000
	PortEndRange   = 35000
)

// GenerateServiceLabels will return the labels that are related to the specified service.
func GenerateServiceLabels(s manifest.Service) map[string]string {
	m := DefaultLabels()
	m[LabelHash] = s.CalculateConfigurationHash()
	m[LabelName] = s.Name
	return m
}

// DefaultLabels will return the default labels that should be put on every object created using the docker API.
func DefaultLabels() map[string]string {
	return map[string]string{
		LabelManagedBy: ApplicationName,
	}
}

// TransformEnvironment will transform the environment variables into a string slice.
func TransformEnvironment(env map[string]string) []string {
	output := make([]string, 0, len(env))
	for k, v := range env {
		output = append(output, k+"="+v)
	}
	return output
}

// GetAvailablePort finds the next available port within a specified range.
func GetAvailablePort(start, end int) int {
	for port := start; port <= end; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close() // Close the listener to free the port
			return port
		}
	}

	return 0
}

// ToBytes will convert the MegaBytes to bytes.
func ToBytes(mb int64) int64 {
	return mb * 1000 * 1000
}
