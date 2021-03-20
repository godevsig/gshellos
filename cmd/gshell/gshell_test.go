package main

import (
	"os"
	"path/filepath"
	"testing"

	gs "github.com/godevsig/gshellos"
)

func TestFiles(t *testing.T) {
	f, err := os.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range names {
		mached, _ := filepath.Match("*_test.gsh", file)
		if mached {
			if err := gs.FileTest(file); err != nil {
				t.Fatalf("%s: %v", file, err)
			}
		}
	}
}
