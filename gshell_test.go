package gshellos_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	gs "github.com/godevsig/gshellos"
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
	err = gs.ShellMain()
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

func randID() string {
	b := make([]byte, 3)
	rand.Read(b)
	id := hex.EncodeToString(b)
	return id
}

func makeCmd(cmdstr string) *exec.Cmd {
	prefix := "-test.run ^TestRunMain$ -test.coverprofile=.test/l2_" + strings.Split(cmdstr, " ")[0] + randID() + ".cov -- "
	return exec.Command("gshell.tester", strings.Split(prefix+cmdstr, " ")...)
}

func gshellRunCmd(cmdstr string) (string, error) {
	cmd := makeCmd(cmdstr)
	out, err := cmd.CombinedOutput()
	return getSout(out), err
}

func gshellRunCmdTimeout(cmdstr string, nSec int) (string, error) {
	cmd := makeCmd(cmdstr)
	file := ".test/tmp_output"
	outFile, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	time.AfterFunc(time.Duration(nSec)*time.Second, func() { cmd.Process.Signal(os.Interrupt) })
	err = cmd.Run()
	var out bytes.Buffer
	outFile.Seek(0, 0)
	out.ReadFrom(outFile)
	return out.String(), err
}

func TestCmdAutoRestart(t *testing.T) {
	out, err := gshellRunCmd("run -group autorestart testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("run -group autorestart testdata/sleep.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("ps")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	pidOld := gs.RunShCmd("ps -eo pid,cmd | grep autorestart | grep -v grep | awk '{print $1}'")
	t.Logf("\n%s", pidOld)
	gs.RunShCmd(fmt.Sprintf("kill -9 %s", pidOld))
	time.Sleep(time.Second)

	pidNew := gs.RunShCmd("ps -eo pid,cmd | grep autorestart | grep -v grep | awk '{print $1}'")
	t.Logf("\n%s", pidNew)

	if pidOld == pidNew {
		t.Fatal("restart grg error")
	}

	out, err = gshellRunCmd("ps")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "running") ||
		!strings.Contains(out, "exited:OK") {
		t.Fatal("unexpected output")
	}
}

func TestCmdRunWrongGRGVer(t *testing.T) {
	out, err := gshellRunCmd("run -group testgrg testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("run -group testgrg-v0.0.0 testdata/hello.go")
	t.Logf("\n%s", out)
	if !strings.Contains(out, "running GRG version v0.0.0 not found") {
		t.Fatal("expected version not found error")
	}

	out, err = gshellRunCmd("ps")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCmdList(t *testing.T) {
	out, err := gshellRunCmd("list")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "godevsig                  gshellDaemon              self          1111") {
		t.Fatal("unexpected output")
	}
}

func TestCmdID(t *testing.T) {
	out, err := gshellRunCmd("id")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 13 {
		t.Fatalf("unexpected lenth: %d", len(out))
	}
}

func TestCmdExec(t *testing.T) {
	out, err := gshellRunCmd("exec testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello, playground\n" {
		t.Fatal("unexpected output")
	}

	out, err = gshellRunCmd("exec testdata/nofile.go")
	t.Logf("\n%s", out)
	if !strings.Contains(out, "testdata/nofile.go not found") {
		t.Fatal("unexpected output")
	}
}

func TestCmdRun(t *testing.T) {
	out, err := gshellRunCmd("run -i testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello, playground\n" {
		t.Fatal("unexpected output")
	}
}

func TestCmdRunRT(t *testing.T) {
	out, err := gshellRunCmd("run -group testrt -rt 50 testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(out)
	out, _ = gshellRunCmdTimeout("log "+id, 1)
	if !strings.Contains(out, "Hello, playground\n") {
		t.Fatal("unexpected output")
	}
}

func TestCmdKill(t *testing.T) {
	out, err := gshellRunCmd("run -group test testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("run -group test2 testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("kill test1 test2")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "none killed") {
		t.Fatal("unexpected output")
	}

	out, err = gshellRunCmd("-v")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	version := strings.TrimSuffix(out, "\n")

	out, err = gshellRunCmd("kill test2*")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "test2-"+version+" killed") {
		t.Fatal("unexpected output")
	}

	out, err = gshellRunCmd("kill -f test-" + version)
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "test-"+version+" killed") {
		t.Fatal("unexpected output")
	}

	out, err = gshellRunCmd("kill test*")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	out, err = gshellRunCmd("ps -group test*")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "test") {
		t.Fatal("unexpected output")
	}
}

func TestCmdPs(t *testing.T) {
	out, err := gshellRunCmd("ps")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GRE ID        IN GROUP            NAME                START AT             STATUS") {
		t.Fatal("unexpected output")
	}
}

func TestCmdPsID(t *testing.T) {
	out, err := gshellRunCmd("run testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	out, err = gshellRunCmd("ps " + strings.TrimSpace(out))
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "NAME      : hello") {
		t.Fatal("unexpected output")
	}
}

func TestCmdStopRm(t *testing.T) {
	out, err := gshellRunCmd("run testdata/sleep.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(out)
	time.Sleep(1 * time.Second)
	out, err = gshellRunCmd("stop " + id)
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stopped") {
		t.Fatal("unexpected output")
	}
	out, err = gshellRunCmd("rm " + id)
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "removed") {
		t.Fatal("unexpected output")
	}
}

func TestCmdStart(t *testing.T) {
	out, err := gshellRunCmd("run testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(out)
	out, err = gshellRunCmd("start " + id)
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "started") {
		t.Fatal("unexpected output")
	}
}

func TestCmdInfo(t *testing.T) {
	out, err := gshellRunCmd("info")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stdbase") {
		t.Fatal("unexpected output")
	}
}

func TestCmdLog(t *testing.T) {
	out, err := gshellRunCmd("run testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(out)
	out, err = gshellRunCmdTimeout("log "+id, 1)
	t.Logf("\n%s", out)
	if !strings.Contains(out, "Hello, playground\n") {
		t.Fatal("unexpected output")
	}
}

func TestCmdRepo(t *testing.T) {
	out, err := gshellRunCmd("repo")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	if out != "github.com/godevsig/gshellos master\n" {
		t.Fatal("unexpected output")
	}

	out, err = gshellRunCmd("run nosuchfile.go")
	t.Logf("\n%s", out)
	if err == nil {
		t.Fatal("expected 404 error or *net/url.Error")
	}
}

func TestCmdRepoRun(t *testing.T) {
	gs.RunShCmd("mv testdata/hello.go testdata/_hello.go")
	defer gs.RunShCmd("mv testdata/_hello.go testdata/hello.go")
	out, err := gshellRunCmd("run -i testdata/hello.go")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello, playground\n" {
		t.Fatal("unexpected output")
	}
}

func TestCmdREPL(t *testing.T) {
	inFile, err := os.Open("testdata/repl.go")
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
	if !strings.Contains(out, "hello 10 times") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestAutoUpdate(t *testing.T) {
	os.WriteFile("bin/rev", []byte("11111111111111111111111111111111\n"), 0644)
	gs.RunShCmd("cp -f bin/gshell.tester bin/gshell." + runtime.GOARCH)
	md5sum := gs.RunShCmd("md5sum bin/gshell." + runtime.GOARCH)
	os.WriteFile("bin/md5sum", []byte(md5sum), 0644)
	t.Logf("\n%s", gs.RunShCmd("cat bin/rev bin/md5sum"))
	oldpid := gs.RunShCmd("pidof gshell.tester")
	t.Logf("\n%s", oldpid)

	out, err := gshellRunCmd("run testdata/fileserver.go -dir bin -port 9001")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}

	id := strings.TrimSpace(out)
	defer func() {
		gs.RunShCmd("rm -f gshell.tester")
		out, err := gshellRunCmd("stop " + id)
		t.Logf("\n%s", out)
		if err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(8 * time.Second)
	pids := gs.RunShCmd("pidof gshell.tester")
	t.Logf("\n%s", pids)
	if strings.Contains(pids, oldpid) {
		t.Fatal("old pid still running")
	}

	out, err = gshellRunCmd("ps")
	t.Logf("\n%s", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "GRE ID        IN GROUP            NAME                START AT             STATUS") {
		t.Fatal("unexpected output")
	}
}

func TestRunMain(t *testing.T) {
	os.Args = append([]string{os.Args[0]}, flag.Args()...)
	err := gs.ShellMain()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	if len(flag.Args()) == 0 {
		cmdstr := "-test.run ^TestRunMain$ -test.coverprofile=.test/l2_gshelld" + randID() + ".cov -- "
		cmdstr += "-wd .working -loglevel debug daemon -registry 127.0.0.1:11985 -bcast 9923 "
		cmdstr += "-root -repo github.com/godevsig/gshellos/master "
		cmdstr += "-update http://127.0.0.1:9001/%s"
		go func() {
			output, _ := exec.Command("gshell.tester", strings.Split(cmdstr, " ")...).CombinedOutput()
			fmt.Println(string(output))
		}()

		time.Sleep(time.Second)
		ret := m.Run() // run tests
		time.Sleep(time.Second)
		exec.Command("pkill", "-SIGINT", "gshell.tester").Run()
		time.Sleep(time.Second)
		os.Exit(ret)
	} else {
		os.Exit(m.Run())
	}
}
