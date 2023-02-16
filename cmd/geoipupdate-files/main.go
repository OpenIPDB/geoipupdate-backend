package main

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/OpenIPDB/geoipupdate-backend/backend"
)

func main() {
	handler := &backend.HTTPHandler{Handler: &Handler{os.DirFS("databases")}}
	_ = http.ListenAndServe("localhost:8080", handler)
}

type Handler struct {
	fs fs.FS
}

func (h *Handler) HomePage() string { return "https://github.com/OpenIPDB" }

func (h *Handler) Login(string, string) error { return nil }

func (h *Handler) ServeMMDB(accountId, editionId string, hash []byte) (payload io.Reader, modified time.Time, err error) {
	database, _ := h.fs.Open(editionId + ".mmdb")
	stat, err := database.Stat()
	if err != nil {
		err = backend.ErrDatabaseNotFound
	} else {
		payload = database
		modified = stat.ModTime()
	}
	return
}
