package gshellos

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func getWant(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	_, err = b.ReadFrom(f)
	if err != nil {
		return "", err
	}

	var s strings.Builder
	start := false
	for {
		line, err := b.ReadString('\n')
		if len(line) != 0 {
			if start {
				if strings.HasPrefix(line, "//") {
					s.WriteString(line[2:])
				}
			}
			if strings.TrimSpace(line) == "//output:" {
				start = true
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return s.String(), nil
}

func getShellMainOutput() (string, error) {
	os.Mkdir(".test", 0755)
	file := ".test/tmp_output"
	outFile, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer func() { outFile.Close() }()

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	os.Stdout = outFile
	err = ShellMain()
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	outFile.Seek(0, 0)
	_, err = b.ReadFrom(outFile)
	return b.String(), err
}

// FileTest is a helper to write extension package UT.
// See extension/shell/shell_test.go and extension/shell/shell_test.gsh.
func FileTest(file string) error {
	want, err := getWant(file)
	if err != nil {
		return fmt.Errorf("failed to get wanted string to compare with: %w", err)
	}

	os.Args = []string{"main", file}
	out, err := getShellMainOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}

	if out != want {
		return fmt.Errorf("want:\n<%s>, got:\n<%s>", want, out)
	}
	return nil
}
