package model

import (
	"bytes"
	"testing"
)

func AssertEquals(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected '%v', got '%v'", expected, actual)
	}
}

func TestParse(t *testing.T) {

	data := `{
	"name": "my-project",
	"services": [
		{
			"name": "website",
			"source": {
				"type": "git",
				"uri": "git@github.com/user/website.git"
			},
			"environment": {
				"ENV_VAR": "one",
				"ANOTHER_VAR": "two"
			},
			"ingress": {
				"domain": "www.example.com",
				"container_port": 80,
				"ssl_challenge": "http01"
			}
		},
		{
			"name": "database",
			"source": {
				"type": "docker",
				"uri": "postgresql:latest"
			}
		}
	]
}`

	reader := bytes.NewBufferString(data)
	project, err := ParseProject(reader)
	if err != nil {
		t.Fatal(err)
	}

	AssertEquals(t, "my-project", project.Name)
	AssertEquals(t, 2, len(project.Services))

	if len(project.Services) != 2 {
		t.FailNow()
	}

	// first service 'website'
	website := project.Services[0]
	AssertEquals(t, "website", website.Name)
	AssertEquals(t, "git", website.Source.Type)
	AssertEquals(t, "git@github.com/user/website.git", website.Source.URI)
	AssertEquals(t, "one", website.Environment["ENV_VAR"])
	AssertEquals(t, "two", website.Environment["ANOTHER_VAR"])
	AssertEquals(t, "www.example.com", website.Ingress.Domain)
	AssertEquals(t, 80, website.Ingress.ContainerPort)
	AssertEquals(t, ChallengeHttp01, website.Ingress.SslChallenge)

	// second service 'database'
	database := project.Services[1]
	AssertEquals(t, "database", database.Name)
	AssertEquals(t, "docker", database.Source.Type)
	AssertEquals(t, "postgresql:latest", database.Source.URI)

}

func TestService_CalculateConfigurationHash(t *testing.T) {
	base := new(Service)
	base.Name = "base"
	base.Source.Type = "docker"
	base.Source.URI = "nginx:latest"
	base.Environment = map[string]string{
		"HTTP_PORT": "80",
		"ANOTHER":   "default-value",
	}

	{
		// with different source URI
		compare := new(Service)
		compare.Name = "base"
		compare.Source.Type = "docker"
		compare.Source.URI = "nginx:0.0.1"
		compare.Environment = base.Environment

		actual := base.CalculateConfigurationHash()
		calculated := compare.CalculateConfigurationHash()
		if actual == calculated {
			t.Errorf("hash should change when the URI is differnt (hash=%s)", actual)
		}
	}

	{
		// with completely different source type
		compare := new(Service)
		compare.Name = "base"
		compare.Source.Type = "git"
		compare.Source.URI = "git@github.com/user/repo"
		compare.Environment = base.Environment

		actual := base.CalculateConfigurationHash()
		calculated := compare.CalculateConfigurationHash()
		if actual == calculated {
			t.Errorf("Hash should change when source is different (hash=%s)", actual)
		}
	}

	{
		// different name should have no impact (configuration stays the same)
		compare := new(Service)
		compare.Name = "another"
		compare.Source.Type = base.Source.Type
		compare.Source.URI = base.Source.URI
		compare.Environment = base.Environment

		actual := base.CalculateConfigurationHash()
		calculated := compare.CalculateConfigurationHash()
		if actual != calculated {
			t.Errorf("Hash should be the same when ONLY name is different (expected=%s, actual=%s)", actual, calculated)
		}
	}

	{
		// environment changes should have an impact
		compare := new(Service)
		compare.Name = "another"
		compare.Source.Type = base.Source.Type
		compare.Source.URI = base.Source.URI
		compare.Environment = map[string]string{
			"HTTP_PORT": "80",
			"ANOTHER":   "different value",
		}

		actual := base.CalculateConfigurationHash()
		calculated := compare.CalculateConfigurationHash()
		if actual == calculated {
			t.Errorf("Hash should differ when environment values are different (hash=%s)", actual)
		}
	}

	{
		// with different container port
		compare := new(Service)
		compare.Name = base.Name
		compare.Source.Type = base.Source.Type
		compare.Source.URI = base.Source.URI
		compare.Environment = base.Environment
		compare.Ingress.ContainerPort = 443

		actual := base.CalculateConfigurationHash()
		calculated := compare.CalculateConfigurationHash()
		if actual == calculated {
			t.Errorf("Hash should differ when container port values are different (hash=%s)", actual)
		}
	}
}

func BenchmarkService_CalculateConfigurationHash(b *testing.B) {
	base := new(Service)
	base.Name = "base"
	base.Source.Type = "docker"
	base.Source.URI = "nginx:latest"
	base.Environment = map[string]string{
		"HTTP_PORT": "80",
		"ANOTHER":   "default-value",
	}

	actual := base.CalculateConfigurationHash()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compare := base.CalculateConfigurationHash()
			if actual != compare {
				b.Fatalf("Hash should be exactly the same (expected=%s, actual=%s)", actual, compare)
			}
		}
	})
}
