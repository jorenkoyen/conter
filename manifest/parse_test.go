package manifest

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
	project, err := Parse(reader)
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
