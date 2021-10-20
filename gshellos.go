package gshellos

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	// ErrBrokenGre is an error where the specified gre has problem to run.
	ErrBrokenGre = errors.New("broken gre")
	// ErrNoUpdate is an error that no update available
	ErrNoUpdate = errors.New("no update available")
)

func genID(width int) string {
	b := make([]byte, width)
	rand.Read(b)
	id := hex.EncodeToString(b)
	return id
}

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

// RunShCmd runs a command and returns its output.
// The command will be running in background and output is discarded
// if it ends with &.
func RunShCmd(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	bg := false
	if fields[len(fields)-1] == "&" {
		bg = true
		fields = fields[:len(fields)-1]
	}
	if _, err := os.Stat(fields[0]); os.IsNotExist(err) {
		fields = append([]string{"-c"}, strings.Join(fields, " "))
	}
	if bg {
		if err := exec.Command("sh", fields...).Start(); err != nil {
			return err.Error()
		}
		return ""
	}
	output, _ := exec.Command("sh", fields...).CombinedOutput()
	return string(output)
}
