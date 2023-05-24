//go:build stdhttp
// +build stdhttp

package gshellos

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type urlInfo struct {
	domain string
	owner  string
	repo   string
	ref    string
	path   string
}

func (ui urlInfo) replacePath(path string) urlInfo {
	nui := ui
	nui.path = path
	return nui
}

type httpHandler interface {
	list() ([]httpFileInfo, error)
	replacePath(path string) httpHandler
	getArchive() ([]byte, error) // in .zip format
}

type githubHandler struct {
	urlInfo
}

var ghpKey string

func init() {
	const sk = "QmVhcmVyIGdocF9BZVFnY0JFTER2WUoxWXNlN2pUVDFxVWFCbElLb24zMzBsb3M="
	data, _ := base64.StdEncoding.DecodeString(sk)
	ghpKey = string(data)
}

func (hdl githubHandler) list() ([]httpFileInfo, error) {
	url := fmt.Sprintf("https://api.%s/repos/%s/%s/contents/%s?ref=%s", hdl.domain, hdl.owner, hdl.repo, hdl.path, hdl.ref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", ghpKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s not found", hdl.path)
	}
	type fileInfo struct {
		FileType    string `json:"type"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		DownloadURL string `json:"download_url"`
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fis []fileInfo
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&fis); err != nil {
		var fi fileInfo
		// should I do a datacpy := make([]byte, len(data)); copy(datacpy, data) ?
		if err := json.NewDecoder(bytes.NewReader(data)).Decode(&fi); err != nil { // test again if it is file
			return nil, nil // ignore the contents
		}
		fis = []fileInfo{fi}
	}

	var hfis []httpFileInfo
	for _, fi := range fis {
		hfi := httpFileInfo{name: fi.Name, path: fi.Path, downloadURL: fi.DownloadURL}
		if fi.FileType == "dir" {
			hfi.isDir = true
		}
		hfis = append(hfis, hfi)
	}
	return hfis, nil
}

func (hdl githubHandler) replacePath(path string) httpHandler {
	return githubHandler{hdl.urlInfo.replacePath(path)}
}

func (hdl githubHandler) getArchive() ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "gshell-http-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	if err := httpDownload(hdl, tmpDir); err != nil {
		return nil, err
	}
	return zipPathToBuffer(tmpDir)
}

type gitlabHandler struct {
	urlInfo
}

func (hdl gitlabHandler) list() ([]httpFileInfo, error) {
	url := fmt.Sprintf("https://%s/api/v4/projects/%s%%2F%s/repository/tree?path=%s&ref=%s", hdl.domain, hdl.owner, hdl.repo, hdl.path, hdl.ref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s not found", hdl.path)
	}

	type fileInfo struct {
		FileType string `json:"type"`
		Name     string `json:"name"`
		Path     string `json:"path"`
	}
	var fis []fileInfo
	if err := json.NewDecoder(resp.Body).Decode(&fis); err != nil {
		return nil, nil // ignore the contents
	}
	if len(fis) == 0 { // it is a single file
		url := fmt.Sprintf("https://%s/api/v4/projects/%s%%2F%s/repository/files/%s?ref=%s",
			hdl.domain, hdl.owner, hdl.repo, strings.ReplaceAll(hdl.path, "/", "%2F"), hdl.ref)
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s not found", hdl.path)
		}
		fis = []fileInfo{{FileType: "blob", Name: path.Base(hdl.path), Path: hdl.path}}
	}
	var hfis []httpFileInfo
	for _, fi := range fis {
		hfi := httpFileInfo{name: fi.Name, path: fi.Path}
		if fi.FileType == "tree" {
			hfi.isDir = true
		} else {
			hfi.downloadURL = fmt.Sprintf("https://%s/%s/%s/-/raw/%s/%s", hdl.domain, hdl.owner, hdl.repo, hdl.ref, fi.Path)
		}
		hfis = append(hfis, hfi)
	}
	return hfis, nil
}

func (hdl gitlabHandler) replacePath(path string) httpHandler {
	return gitlabHandler{hdl.urlInfo.replacePath(path)}
}

func (hdl gitlabHandler) getArchive() ([]byte, error) {
	url := fmt.Sprintf("https://%s/api/v4/projects/%s%%2F%s/repository/archive.zip?path=%s&sha=%s", hdl.domain, hdl.owner, hdl.repo, hdl.path, hdl.ref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s not found", hdl.path)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "gshell-http-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := unzipBufferToPath(data, tmpDir); err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(tmpDir, "*", hdl.path))
	if err != nil || len(matches) != 1 {
		return nil, errors.New("wrong file tree")
	}
	filePath := matches[0]
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if fi.Mode().IsRegular() {
		filePath = filepath.Dir(filePath)
	}
	return zipPathToBuffer(filePath)
}

func parseURL(url string) (handler httpHandler, err error) {
	urlFields := strings.Split(url, "/")
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("url %s format error", url)
		}
	}()
	switch {
	case strings.Contains(urlFields[2], "github"):
		ui := urlInfo{
			domain: urlFields[2],
			owner:  urlFields[3],
			repo:   urlFields[4],
			ref:    urlFields[6],
			path:   strings.Join(urlFields[7:], "/"),
		}
		return githubHandler{ui}, nil
	case strings.Contains(urlFields[2], "gitlab"):
		ui := urlInfo{
			domain: urlFields[2],
			owner:  urlFields[3],
			repo:   urlFields[4],
			ref:    urlFields[7],
			path:   strings.Join(urlFields[8:], "/"),
		}
		return gitlabHandler{ui}, nil
	}
	return nil, fmt.Errorf("rest api for %s unsupported", urlFields[2])
}

type httpOperationImpl struct{}

func (httpOperationImpl) list(url string) ([]httpFileInfo, error) {
	hdl, err := parseURL(url)
	if err != nil {
		return nil, err
	}
	return hdl.list()
}

func httpDownload(hdl httpHandler, dstDir string) error {
	os.MkdirAll(dstDir, 0755)
	hfis, err := hdl.list()
	if err != nil {
		return err
	}

	dldFile := func(hfi httpFileInfo) error {
		if hfi.downloadURL == "" {
			return fmt.Errorf("unknown download URL for %s", hfi.name)
		}
		resp, err := http.Get(hfi.downloadURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s not found", hfi.name)
		}

		oFile, err := os.Create(filepath.Join(dstDir, hfi.name))
		if err != nil {
			return err
		}
		defer oFile.Close()
		if _, err := io.Copy(oFile, resp.Body); err != nil {
			return err
		}
		return nil
	}

	for _, hfi := range hfis {
		if hfi.isDir {
			if err := httpDownload(hdl.replacePath(hfi.path), filepath.Join(dstDir, hfi.name)); err != nil {
				return err
			}
		} else {
			if err := dldFile(hfi); err != nil {
				return err
			}
		}
	}
	return nil
}

func (httpOperationImpl) download(url string, dstDir string) error {
	hdl, err := parseURL(url)
	if err != nil {
		return err
	}
	return httpDownload(hdl, dstDir)
}

func (httpOperationImpl) readFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("url %s not found: %d error", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (httpOperationImpl) getArchive(url string) ([]byte, error) {
	hdl, err := parseURL(url)
	if err != nil {
		return nil, err
	}
	return hdl.getArchive()
}

func init() {
	httpOp = httpOperationImpl{}
}
