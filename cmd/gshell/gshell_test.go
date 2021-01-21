package main

import (
	"testing"

	gs "github.com/godevsig/gshellos"
)

func TestFile(t *testing.T) {
	err := gs.FileTest("gshell_test.gsh")
	if err != nil {
		t.Fatal(err)
	}
}
