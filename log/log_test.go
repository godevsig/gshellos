package log

import (
	"bufio"
	"fmt"
	stdlog "log"
	"os"
	"path"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

func (lg *Logger) normalLevelPrintf(format string, args ...interface{}) {
	lg.Tracef(format, args...)
	lg.Debugf(format, args...)
	lg.Infof(format, args...)
	lg.Warnf(format, args...)
	lg.Errorf(format, args...)
}

func (lg *Logger) normalLevelPrintln(args ...interface{}) {
	lg.Traceln(args...)
	lg.Debugln(args...)
	lg.Infoln(args...)
	lg.Warnln(args...)
	lg.Errorln(args...)
}

func ExampleLogger_Infoln() {
	lg := DefaultStream.NewLogger("demo", Linfo)
	defer lg.Close()
	// recreate the same name logger will panic
	//lg = DefaultStream.NewLogger("demo")
	DefaultStream.SetTimeFormat("")
	lg.Infoln("no need to put a new line")
	lg.Infoln("another line")

	//Output:
	//[demo][INFO] no need to put a new line
	//[demo][INFO] another line
}

func ExampleDefaultStream_NewLogger() {
	lg := DefaultStream.NewLogger("example", Linfo)
	defer lg.Close()
	DefaultStream.SetTimeFormat("")
	DefaultStream.SetFlag(Ldefault)
	i := 100
	lg.normalLevelPrintf("hello world %d", i)
	lg.normalLevelPrintln("ni hao", i+1)
	DefaultStream.SetLoglevel("*", Ldebug)
	lg.normalLevelPrintf("hello world %d", i+2)
	lg.normalLevelPrintln("ni hao", i+3)

	//Output:
	//
	//[example][INFO] hello world 100
	//[example][WARN] hello world 100
	//[example][ERROR] hello world 100
	//[example][INFO] ni hao 101
	//[example][WARN] ni hao 101
	//[example][ERROR] ni hao 101
	//[example][DEBUG] hello world 102
	//[example][INFO] hello world 102
	//[example][WARN] hello world 102
	//[example][ERROR] hello world 102
	//[example][DEBUG] ni hao 103
	//[example][INFO] ni hao 103
	//[example][WARN] ni hao 103
	//[example][ERROR] ni hao 103
}

func ExampleStream_NewLogger() {
	// in your main.go
	stream := NewStream("mystream")
	defer stream.Close()
	stream.SetTimeFormat("")

	// in your xxxpackage.go
	lg := GetStream("mystream").NewLogger("myexample", Linfo)
	defer lg.Close()
	i := 200
	lg.normalLevelPrintf("hello world %d", i)

	// if you need to change loglevel in elsewhere.go
	GetStream("mystream").SetLoglevel("*", Ltrace)
	lg.normalLevelPrintf("hello world %d", i+1)

	//Output:
	//[myexample][INFO] hello world 200
	//[myexample][WARN] hello world 200
	//[myexample][ERROR] hello world 200
	//[myexample][TRACE] hello world 201
	//[myexample][DEBUG] hello world 201
	//[myexample][INFO] hello world 201
	//[myexample][WARN] hello world 201
	//[myexample][ERROR] hello world 201
}

func ExampleStream_SetLoglevel() {
	stream := NewStream("examplePattern")
	defer stream.Close()
	stream.SetTimeFormat("")
	lg := stream.NewLogger("service", Linfo)
	defer lg.Close()
	var wg sync.WaitGroup

	type workerInfo struct {
		workerName string
		workerlg   *Logger
	}

	freeInstance := func(wi *workerInfo, id int) {
		wi.workerlg.Debugln("free instance:", id)
	}

	handleworker := func(wi *workerInfo) {
		wi.workerlg.Infoln("doing something awaresome...")
		freeInstance(wi, 3)
		wi.workerlg.Infoln("work done")
		wi.workerlg.Close()
		lg.Infof("worker: %s closed", wi.workerName)
		wg.Done()
	}

	onworkerDetect := func(workerName string) {
		lg.Infof("creating a new routine to handle %s", workerName)
		wi := &workerInfo{workerName: workerName, workerlg: stream.NewLogger(fmt.Sprintf("service/%s", workerName), Linfo)}
		lg.Infoln("handling worker info...")
		go handleworker(wi)
	}

	wg.Add(3)
	//with pattern "service/worker123*", worker123456 and worker123999 match and are set to Ldebug level
	stream.SetLoglevel("service/worker123*", Ldebug)
	onworkerDetect("worker123456")
	onworkerDetect("worker123999")
	onworkerDetect("worker555666")

	wg.Wait()

	//Unordered output:
	//[service][INFO] creating a new routine to handle worker123456
	//[service][INFO] handling worker info...
	//[service][INFO] creating a new routine to handle worker123999
	//[service][INFO] handling worker info...
	//[service][INFO] creating a new routine to handle worker555666
	//[service][INFO] handling worker info...
	//[service/worker555666][INFO] doing something awaresome...
	//[service/worker555666][INFO] work done
	//[service][INFO] worker: worker555666 closed
	//[service/worker123999][INFO] doing something awaresome...
	//[service/worker123999][DEBUG] free instance: 3
	//[service/worker123999][INFO] work done
	//[service][INFO] worker: worker123999 closed
	//[service/worker123456][INFO] doing something awaresome...
	//[service/worker123456][DEBUG] free instance: 3
	//[service/worker123456][INFO] work done
	//[service][INFO] worker: worker123456 closed
}

func ExampleStream_SetOutput() {
	mystream := NewStream("mystream")
	defer mystream.Close()
	//set log output to file, args with "file:<file path>"
	mystream.SetOutput("file:test/test.log")
	mystream.SetTimeFormat(time.RFC3339Nano)
}

type nullFactory struct{}
type null struct{}

func (nullFactory) newOutputter(description string) (outputter, error) {
	return null{}, nil
}
func (null) output(*logEntry) {}

func (null) Write(buf []byte) (int, error) { return len(buf), nil }

func init() {
	newOutputterFactory("null", nullFactory{})
}

func TestPatternMatch(t *testing.T) {
	cases := []struct {
		str, pattern string
		match        bool
	}{
		{"service/worker1234", "*", true},
		{"service/worker1234", "service/*", true},
		{"service/worker1234", "service/worker1234", true},
		{"service/worker1234", "service/worker45", false},
		{"service/worker1234", "service/worker12*", true},
		{"111foobar555", "*foo*bar*", true},
		{"111foobar555", "foo*bar*", false},
		{"111foobar555", "*foo*bar", false},
		{"foobar555", "*foo*bar", false},
		{"foobar555", "*foo*bar*", true},
		{"foobar", "bar", false},
		{"foobar", "bar*", false},
		{"111foobar555", "*bar*foo*", false},
		{"1111foobar555", "111*foo*foo*", false},
		{"1111foobarfoo555", "111*foo*foo*5555", false},
		{"1111foobarfoo555", "111*foo*foo*555", true},
		{"foobarfoo555", "*bar*555", true},
		{"foobarfoo5555", "*bar*555", true},
	}

	for _, c := range cases {
		match := patternMatch(c.str, c.pattern)
		if match != c.match {
			t.Errorf("if %s matches %s: Got %v, want %v", c.pattern, c.str, match, c.match)
		}
	}
}

func TestFileLogger(t *testing.T) {
	file := "test/test.log"
	os.Remove(file)
	dst := fmt.Sprintf("file:%s", file)

	testStream := NewStream("test")
	defer testStream.Close()
	if err := testStream.SetOutput(dst); err != nil {
		t.Fatal(err)
	}
	testStream.SetFlag(Lfileline)
	lg := testStream.NewLogger("testfile", Linfo)
	defer lg.Close()

	msg := "hello world"
	lg.Infoln(msg)
	testStream.SetOutput("stdout")

	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	line, err := rd.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	pos := len("[2020/08/10 03:30:30.712161][testfile][INFO](log_test.go:108) ")
	if line[pos:pos+len(msg)] != msg {
		t.Fatalf("unexpected msg: got %s, expect %s", line[pos:pos+len(msg)], msg)
	}
}

func TestFileLoggerConcurrency(t *testing.T) {
	testStream := NewStream("testconcurrency")
	defer testStream.Close()
	lg := testStream.NewLogger("testfile", Linfo)
	defer lg.Close()
	file := "test/testc.log"

	doTest := func(t *testing.T, msg string) {
		os.Remove(file)
		dst := fmt.Sprintf("file:%s", file)
		if err := testStream.SetOutput(dst); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		routineNum := 100
		lineNum := 100
		wg.Add(routineNum)
		for i := 0; i < routineNum; i++ {
			go func() {
				for i := 0; i < lineNum; i++ {
					lg.Infoln(msg)
				}
				wg.Done()
			}()
		}
		wg.Wait()
		testStream.SetOutput("stdout")

		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		var i int
		pos := len("[2020/08/09 14:47:41.769979][testfile][INFO] ")
		expectLen := pos + len(msg)
		for i = 0; scanner.Scan(); i++ {
			line := scanner.Text()
			if len(line) != expectLen {
				break
			}

			if scanner.Text()[pos:expectLen] != msg {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if i != routineNum*lineNum {
			t.Fatalf("Unexpected line number: got %d, expect %d", i, routineNum*lineNum)
		}
	}

	t.Run("short msg", func(t *testing.T) { doTest(t, "hello world") })

	msg := ""
	for i := 0; i < 1000; i++ {
		msg = msg + fmt.Sprintf("ni hao %d ", i)
	}
	t.Run("long msg", func(t *testing.T) { doTest(t, msg) })
}

func TestEmptyOutput(t *testing.T) {
	lg := DefaultStream.NewLogger("testEmpty", Linfo)
	defer lg.Close()
	lg.Infof("%s", "")
}

func TestLoggerNames(t *testing.T) {
	lgns := []string{"test1", "www", "test2", "test3", "we1", "xw2", "test4"}

	testStream := NewStream("testnames")
	defer testStream.Close()
	for _, lgn := range lgns {
		testStream.NewLogger(lgn, Linfo)
	}
	algns := testStream.AllLoggerNames()
	sort.Strings(algns)
	sort.Strings(lgns)

	if !reflect.DeepEqual(algns, lgns) {
		t.Fatalf("Got %v, expect %v", algns, lgns)
	}

	for _, lgn := range algns {
		lg := testStream.GetLogger(lgn)
		if lg == nil {
			t.Fatalf("Get logger instance %s failed", lgn)
		}
		lg.Close()
	}

	for _, lgn := range algns {
		lg := testStream.GetLogger(lgn)
		if lg != nil {
			t.Fatalf("Logger instance %s should have been closed", lgn)
		}
	}
}

func TestFatalPrintf(t *testing.T) {
	lg := DefaultStream.NewLogger("testFatalf", Linfo)
	defer lg.Close()
	defer func() {
		if err := recover(); err != nil {
			t.Log(err)
		}
	}()

	lg.Fatalf("Fatal will panic with: %s", "last message")
	t.Fatal("never reach here, fatal api should panic")
}

func TestFatalPrintln(t *testing.T) {
	lg := DefaultStream.NewLogger("testFatalln", Linfo)
	defer lg.Close()
	defer func() {
		if err := recover(); err != nil {
			t.Log(err)
		}
	}()

	lg.Fatalln("Fatal will panic with:", "last message")
	t.Fatal("never reach here, fatal api should panic")
}

func BenchmarkNullLogger(b *testing.B) {
	testStream := NewStream("test")
	defer testStream.Close()
	lg := testStream.NewLogger("Benchmark", Linfo)

	testStream.SetOutput("null")
	testStream.SetFlag(Ldefault)
	testStream.SetTimeFormat("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Infoln("hello world")
	}
}

func BenchmarkNullLoggerTimeStamp(b *testing.B) {
	testStream := NewStream("test")
	defer testStream.Close()
	lg := testStream.NewLogger("Benchmark", Linfo)

	testStream.SetOutput("null")
	testStream.SetFlag(Ldefault)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Infoln("hello world")
	}
}

func BenchmarkNullStdLogger(b *testing.B) {
	lg := stdlog.New(null{}, "test ", stdlog.LstdFlags)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Println("hello world")
	}
}

func BenchmarkNullLoggerFileline(b *testing.B) {
	testStream := NewStream("test")
	defer testStream.Close()
	lg := testStream.NewLogger("Benchmark", Linfo)

	testStream.SetOutput("null")
	testStream.SetFlag(Lfileline)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Infoln("hello world")
	}
}

func BenchmarkFileLogger(b *testing.B) {
	testStream := NewStream("test")
	defer testStream.Close()
	lg := testStream.NewLogger("Benchmark", Linfo)

	if err := testStream.SetOutput("file:test/test.log"); err != nil {
		b.Fatal(err)
	}
	//testStream.SetFlag(Lfileline)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Infoln("hello world")
	}
}

func BenchmarkFileStdLogger(b *testing.B) {
	filename := "test/test.log"
	dir := path.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		b.Fatal(err)
	}

	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		b.Fatal()
	}

	lg := stdlog.New(f, "test ", stdlog.LstdFlags)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Println("hello world")
	}
}
