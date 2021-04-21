package gshellos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	os.Mkdir(".test", 0755)
}

func prepareSrc(src string) string {
	file := ".test/main_test.gsh"
	out, err := os.Create(file)
	if err != nil {
		return ""
	}
	defer out.Close()

	out.Write([]byte(src))
	return file
}

func doTestSrc(src string) (string, error) {
	file := prepareSrc(src)
	os.Args = []string{"main", file}
	return getShellMainOutput()
}

func TestWrongUsage(t *testing.T) {
	type wrongCases struct {
		input      []string
		wantErrMsg string
	}
	wCases := []wrongCases{
		{
			[]string{"main", "-c", "xxxx.wrong"},
			"wrong file suffix",
		},
		{
			[]string{"main", "-c"},
			"no file provided",
		},
	}

	for _, c := range wCases {
		os.Args = c.input
		err := ShellMain()
		if err == nil {
			t.Fatalf("Expected: %s", c.wantErrMsg)
		}

		if !strings.Contains(err.Error(), c.wantErrMsg) {
			t.Errorf("Expected: %s, found: %s", c.wantErrMsg, err.Error())
		}
	}
}

func TestHelloWorld(t *testing.T) {
	src := `
fmt := import("fmt")
fmt.println("hello world")
`
	out, err := doTestSrc(src)
	if err != nil {
		t.Fatal(err)
	}
	want := `hello world
`
	if out != want {
		t.Fatalf("Want:\n%s, Got:\n%s", want, out)
	}
}

func TestParseError(t *testing.T) {
	src := `
fmt := import("fmt")
v := notDefined
`
	wantErrMsg := "unresolved reference"
	_, err := doTestSrc(src)
	if err == nil {
		t.Fatalf("Expected: %s", wantErrMsg)
	}

	if !strings.Contains(err.Error(), wantErrMsg) {
		t.Errorf("Expected: %s, found: %s", wantErrMsg, err.Error())
	}
}

func TestSaveRun(t *testing.T) {
	src := `
fmt := import("fmt")
fmt.println("hello world")
`
	file := prepareSrc(src)
	os.Args = []string{"main", "-c", file}

	if err := ShellMain(); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"main", strings.TrimSuffix(file, filepath.Ext(file))}
	out, err := getShellMainOutput()
	if err != nil {
		t.Fatal(err)
	}
	want := `hello world
`
	if out != want {
		t.Fatalf("Want:\n%s, Got:\n%s", want, out)
	}
}

func TestREPL(t *testing.T) {
	src := `
fmt := import("fmt")
fmt.println("hello world")
help(ex("hello world").split().join(", "))
`
	file := prepareSrc(src)
	inFile, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer inFile.Close()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = inFile

	os.Args = []string{"main"}
	out, err := getShellMainOutput()
	if err != nil {
		t.Fatal(err)
	}
	want := `>> >> >> hello world
>> hello, world
>> `
	if out != want {
		t.Fatalf("Want:\n%s, Got:\n%s", want, out)
	}
}

func TestRunMain(t *testing.T) {
	//fmt.Println(os.Args)
	if len(os.Args) >= 5 {
		args := []string{"main"}
		args = append(args, os.Args[4:]...)

		os.Args = args
		if err := ShellMain(); err != nil {
			t.Fatal(err)
		}
	}
}
