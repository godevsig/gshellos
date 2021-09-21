package gshellos

import (
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"
)

var (
	// ErrBrokenGre is an error where the specified gre has problem to run.
	ErrBrokenGre = errors.New("broken gre")
)

type endlessReader struct {
	r io.Reader
}

func (er endlessReader) Read(p []byte) (n int, err error) {
	for i := 0; i < 30; i++ {
		n, err = er.r.Read(p)
		if err != io.EOF {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	p[n] = 0 // fake read
	return n + 1, nil
}

type writerStat struct {
	w   io.Writer
	eof bool
}
type mWriter struct {
	writers []*writerStat
	eofNum  int
}

func (mw *mWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		if w.eof {
			continue
		}
		if _, err := w.w.Write(p); err != nil {
			w.eof = true
			mw.eofNum++
		}
	}
	if mw.eofNum == len(mw.writers) {
		return 0, io.EOF
	}
	return len(p), nil
}

// multiWriter is like io.MultiWriter but only stops if all writers are EOF.
func multiWriter(writers ...io.Writer) io.Writer {
	mw := &mWriter{}
	for _, w := range writers {
		mw.writers = append(mw.writers, &writerStat{w: w})
	}
	return mw
}

type null struct{}

func (null) Close() error                  { return nil }
func (null) Write(buf []byte) (int, error) { return len(buf), nil }
func (null) Read(buf []byte) (int, error)  { return 0, io.EOF }

// RunCmd runs a command and returns its output
func RunCmd(cmd string) string {
	strs := strings.Split(cmd, " ")
	var cmdStrs []string
	for _, str := range strs {
		if len(str) != 0 {
			cmdStrs = append(cmdStrs, str)
		}
	}
	if len(cmdStrs) == 0 {
		return ""
	}
	name := cmdStrs[0]
	var args []string
	if len(cmdStrs) > 1 {
		args = cmdStrs[1:]
	}

	output, err := exec.Command(name, args...).Output()
	outputStr := string(output)
	if err != nil {
		outputStr = outputStr + err.Error()
	}
	return outputStr
}
