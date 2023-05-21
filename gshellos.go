package gshellos

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	as "github.com/godevsig/adaptiveservice"
)

var (
	// ErrBrokenGRG is an error where the specified GRG has problem to run.
	ErrBrokenGRG = errors.New("broken GRG")
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

type nullIO struct{}

func (nullIO) Close() error                  { return nil }
func (nullIO) Write(buf []byte) (int, error) { return len(buf), nil }
func (nullIO) Read(buf []byte) (int, error)  { return 0, io.EOF }

func init() {
	as.RegisterType((*net.DNSError)(nil))
	as.RegisterType((*net.OpError)(nil))
	as.RegisterType((*net.TCPAddr)(nil))
	as.RegisterType((*os.SyscallError)(nil))
	as.RegisterType(syscall.Errno(0))
	as.RegisterType((*url.Error)(nil))
	as.RegisterType((*exec.ExitError)(nil))
	as.RegisterType((*fs.PathError)(nil))
}

type httpFileInfo struct {
	name        string
	path        string
	downloadURL string
	isDir       bool
}

type httpOperation interface {
	// list directory contents
	list(url string) ([]httpFileInfo, error)
	// download all contents recursively
	download(url string, dstDir string) error
	// read file content
	readFile(url string) ([]byte, error)
	// get .zip archive
	getArchive(url string) ([]byte, error)
}

var httpOp httpOperation

// save folder in .zip format
func zipPathToBuffer(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var data bytes.Buffer
	zw := zip.NewWriter(&data)

	if fi.Mode().IsRegular() {
		fileName := filepath.Base(path)
		zipEntry, err := zw.Create(fileName)
		if err != nil {
			return nil, err
		}
		// Open the source file
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Copy the file content to the ZIP entry
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return nil, err
		}

		zw.Close()
		return data.Bytes(), nil
	}

	srcDir := path

	// Walk through the source directory
	err = filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Create a new file entry in the ZIP archive
		relPath := strings.TrimPrefix(filePath, srcDir)
		zipEntry, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		// Open the source file
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy the file content to the ZIP entry
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return err
		}

		return nil
	})

	zw.Close()
	return data.Bytes(), err
}

// unzip data to folder
func unzipBufferToPath(data []byte, dstDir string) error {
	r := bytes.NewReader(data)
	zr, err := zip.NewReader(r, r.Size())
	if err != nil {
		return err
	}

	extractFile := func(file *zip.File, dstDir string) error {
		filePath := filepath.Join(dstDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return err
			}

			src, err := file.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, src)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, file := range zr.File {
		err := extractFile(file, dstDir)
		if err != nil {
			return err
		}
	}

	return nil
}
