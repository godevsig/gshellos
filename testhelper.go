package gshellos

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
)

type want struct {
	content   []string
	unordered bool
}

func (wt *want) String() string {
	var sb strings.Builder
	if wt.unordered {
		sb.WriteString("//unordered output:\n")
	} else {
		sb.WriteString("//output:\n")
	}
	for _, str := range wt.content {
		sb.WriteString(str + "\n")
	}
	return sb.String()
}

func getWant(file string) ([]*want, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var b bytes.Buffer
	_, err = b.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	var wants []*want
	var wt *want

	for {
		line, err := b.ReadString('\n')
		if len(line) != 0 {
			line = strings.TrimSpace(line)
			switch line {
			case "//output:":
				wt = &want{}
				wants = append(wants, wt)
			case "//unordered output:":
				wt = &want{unordered: true}
				wants = append(wants, wt)
			default:
				if wt != nil {
					if strings.HasPrefix(line, "//") {
						wt.content = append(wt.content, line[2:])
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	for _, wt := range wants {
		if wt.unordered {
			sort.Strings(wt.content)
		}
	}
	return wants, nil
}

func getShellMainOutput() (string, error) {
	os.Mkdir(".test", 0755)
	file := ".test/tmp_output"
	outFile, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

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
	wants, err := getWant(file)
	if err != nil {
		return fmt.Errorf("failed to get wanted string to compare with: %w", err)
	}

	os.Args = []string{"main", file}
	out, err := getShellMainOutput()
	if err != nil {
		return fmt.Errorf("%w, got output: %s", err, string(out))
	}

	target := bytes.NewBufferString(out)
	for _, wt := range wants {
		var content []string
		for i := 0; i < len(wt.content); i++ {
			line, _ := target.ReadString('\n')
			content = append(content, strings.TrimSpace("//" + line)[2:])
		}
		if wt.unordered {
			sort.Strings(content)
		}
		if !reflect.DeepEqual(content, wt.content) {
			target.WriteString("unmatch")
			break
		}
	}
	if target.Len() != 0 {
		return fmt.Errorf("want(internal structure):\n<%v>, got(raw):\n<%s>", wants, out)
	}

	return nil
}
