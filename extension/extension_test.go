package extension

import (
	"testing"

	gs "github.com/godevsig/gshellos"
)

func TestFile(t *testing.T) {
	err := gs.FileTest("extension_test.gsh")
	if err != nil {
		t.Fatal(err)
	}
}
