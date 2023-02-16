package backend

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidEditionId = &Error{StatusCode: http.StatusNotFound, Message: "Invalid Edition ID"}
	ErrInvalidHash      = &Error{StatusCode: http.StatusNotFound, Message: "Invalid Hash"}
	ErrUnauthorized     = &Error{StatusCode: http.StatusUnauthorized}
	ErrMethodNotAllowed = &Error{StatusCode: http.StatusMethodNotAllowed}
	ErrDatabaseNotFound = &Error{StatusCode: http.StatusNotFound, Message: "Database Not Found"}
	ErrDatabaseUpToDate = &Error{StatusCode: http.StatusNotModified, Message: "Database is Up-to-date"}
)

const (
	prefix = "/geoip/databases/"
)

type Handler interface {
	HomePage() string
	Login(accountId, licenseKey string) error
	ServeMMDB(accountId, editionId string, hash []byte) (io.Reader, time.Time, error)
}

type HTTPHandler struct{ Handler }

func (h *HTTPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Redirect(rw, r, h.HomePage(), http.StatusTemporaryRedirect)
		return
	}
	gzipped, modified, md5Hash, err := h.execute(r)
	if err != nil {
		code := http.StatusInternalServerError
		if e, ok := err.(*Error); ok {
			code = e.StatusCode
		}
		http.Error(rw, err.Error(), code)
	} else {
		header := rw.Header()
		header.Set("Content-Encoding", "gzip")
		header.Set("Last-Modified", modified.Format(time.RFC1123))
		header.Set("X-Database-MD5", hex.EncodeToString(md5Hash))
		_, _ = gzipped.WriteTo(rw)
	}
}

func (h *HTTPHandler) execute(r *http.Request) (gzipped bytes.Buffer, modified time.Time, md5Hash []byte, err error) {
	if r.Method != http.MethodGet {
		err = ErrMethodNotAllowed
		return
	}
	hash, _ := hex.DecodeString(r.URL.Query().Get("db_md5"))
	editionId := getEditionId(r.URL.Path)
	accountId, licenseKey, _ := r.BasicAuth()
	if len(hash) != 16 {
		err = ErrInvalidHash
	} else if editionId == "" {
		err = ErrInvalidEditionId
	}
	if err == nil {
		err = h.Login(accountId, licenseKey)
	}
	if err != nil {
		return
	}
	payload, modified, err := h.Handler.ServeMMDB(accountId, editionId, hash)
	if payload == nil || modified.IsZero() {
		err = ErrDatabaseNotFound
	}
	if err != nil {
		return
	}
	md5HashWriter := md5.New()
	gzipWriter := gzip.NewWriter(&gzipped)
	_, _ = io.Copy(io.MultiWriter(gzipWriter, md5HashWriter), payload)
	_ = gzipWriter.Close()
	md5Hash = md5HashWriter.Sum(nil)
	if bytes.EqualFold(md5Hash, hash) {
		err = ErrDatabaseUpToDate
	}
	return
}

func getEditionId(pathname string) (editionId string) {
	editionId = pathname[len(prefix):]
	editionId = pathname[:strings.IndexRune(editionId, '/')]
	editionId, _ = url.PathUnescape(editionId)
	return
}
