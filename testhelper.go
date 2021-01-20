package gshellos

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func getSout(out []byte) string {
	b := bytes.NewBuffer(out)
	var lines []string
	for {
		line, err := b.ReadString('\n')
		if len(line) != 0 {
			lines = append(lines, line)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}
	}
	if len(lines) <= 2 {
		return ""
	}

	return strings.Join(lines[:len(lines)-2], "")
}

// FileTest is a helper to write extension package UT.
// See extension/shell/shell_test.go and extension/shell/shell_test.gsh.
func FileTest(file string) error {
	want, err := getWant(file)
	if err != nil {
		return fmt.Errorf("failed to get wanted string to compare with: %w", err)
	}

	covFile := filepath.Base(strings.TrimSuffix(file, filepath.Ext(file)))
	covFileArg := fmt.Sprintf("-test.coverprofile=.test/l2_%s.cov", covFile)
	cmd := exec.Command("gshell.test", "-test.run", `^TestRunMain$`, covFileArg, file)
	//fmt.Println(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}

	sout := getSout(out)
	if sout != want {
		return fmt.Errorf("want:\n<%s>, got:\n<%s>", want, sout)
	}
	return nil
}
