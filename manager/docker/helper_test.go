package docker

import "testing"

func TestToBytes(t *testing.T) {

	bytes := ToBytes(128)
	if bytes != 128000000 {
		t.Fatalf("expected 128000000 but got %d", bytes)
	}
}
