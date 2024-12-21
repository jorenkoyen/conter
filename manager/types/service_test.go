package types

import "testing"

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

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
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

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
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

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
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

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
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

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
		if actual == calculated {
			t.Errorf("Hash should differ when container port values are different (hash=%s)", actual)
		}
	}

	{
		// with volumes
		compare := new(Service)
		compare.Name = base.Name
		compare.Source.Type = base.Source.Type
		compare.Source.URI = base.Source.URI
		compare.Environment = base.Environment
		compare.Volumes = []Volume{
			{
				Name: "absolute",
				Path: "/root",
			},
		}

		actual := CalculateHash(base)
		calculated := CalculateHash(compare)
		if actual == calculated {
			t.Errorf("Hash should differ when volumes are defined (hash=%s)", actual)
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

	actual := CalculateHash(base)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compare := CalculateHash(base)
			if actual != compare {
				b.Fatalf("Hash should be exactly the same (expected=%s, actual=%s)", actual, compare)
			}
		}
	})
}
