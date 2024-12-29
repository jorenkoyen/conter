package manager

import (
	"fmt"
	"github.com/jorenkoyen/conter/manager/types"
	"testing"
)

func createEmptyApplyProjectOptions() *ApplyProjectOptions {
	opts := new(ApplyProjectOptions)
	opts.Services = make([]struct {
		Name           string              `json:"name"`
		Source         types.Source        `json:"source"`
		Environment    map[string]string   `json:"environment"`
		IngressDomains []string            `json:"ingress_domains"`
		ContainerPort  int                 `json:"container_port"`
		Volumes        []types.Volume      `json:"volumes"`
		ChallengeType  types.ChallengeType `json:"challenge_type"`
		Quota          types.Quota         `json:"quota"`
	}, 1)
	return opts
}

func AssertNotEmpty(t *testing.T, val string, message string) {
	t.Helper()
	if val == "" {
		t.Error(message)
	}
}

func AssertErrorThrownForField(t *testing.T, err *types.ValidationError, field string) {
	t.Helper()
	if err == nil {
		t.Errorf("Exepcted an error to be thrown when validation for field=%s", field)
	} else {
		reason := err.Reasons[field]
		msg := fmt.Sprintf("Expected a validation error to be present for field=%s", field)
		AssertNotEmpty(t, reason, msg)
	}
}

func TestApplyProjectOptions_validate(t *testing.T) {

	{
		// no project name given
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = ""

		err := opts.validate()
		AssertErrorThrownForField(t, err, "project_name")
	}

	{
		// no services given
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services = nil

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services")
	}

	{
		// no service name given
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = ""

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].name")
	}

	{
		// no source type given
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].source.type")
	}

	{
		// no source uri given
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].source.uri")
	}

	{
		// source type 'git' is not supported
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "git"
		opts.Services[0].Source.URI = "git@github.com/user/repository:master"

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].source.type")

	}

	{
		// source type 'other' is not supported
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "other"
		opts.Services[0].Source.URI = "../"

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].source.type")
	}

	{
		// no container port supplied
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"
		opts.Services[0].Source.URI = "nginx:latest"
		opts.Services[0].IngressDomains = []string{"www.localtest.me"}
		opts.Services[0].ChallengeType = types.ChallengeTypeHTTP

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].container_port")
	}

	{
		// challenge type 'dns' is not supported
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"
		opts.Services[0].Source.URI = "nginx:latest"
		opts.Services[0].IngressDomains = []string{"www.localtest.me"}
		opts.Services[0].ChallengeType = types.ChallengeTypeDNS
		opts.Services[0].ContainerPort = 80

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].challenge_type")
	}

	{
		// challenge type 'tls' is not supported
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"
		opts.Services[0].Source.URI = "nginx:latest"
		opts.Services[0].IngressDomains = []string{"www.localtest.me"}
		opts.Services[0].ChallengeType = types.ChallengeTypeTLS
		opts.Services[0].ContainerPort = 80

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].challenge_type")
	}

	{
		// challenge type 'random' is not supported
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"
		opts.Services[0].Source.URI = "nginx:latest"
		opts.Services[0].IngressDomains = []string{"www.localtest.me"}
		opts.Services[0].ChallengeType = "other"
		opts.Services[0].ContainerPort = 80

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].challenge_type")
	}

	{
		// quota is below 128MB
		opts := createEmptyApplyProjectOptions()
		opts.ProjectName = "default"
		opts.Services[0].Name = "www"
		opts.Services[0].Source.Type = "docker"
		opts.Services[0].Source.URI = "nginx:latest"
		opts.Services[0].IngressDomains = []string{"www.localtest.me"}
		opts.Services[0].ChallengeType = types.ChallengeTypeHTTP
		opts.Services[0].ContainerPort = 80
		opts.Services[0].Quota.MemoryLimit = 64

		err := opts.validate()
		AssertErrorThrownForField(t, err, "services[0].quota.memory_limit")
	}
}
