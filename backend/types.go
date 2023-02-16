package backend

import (
	"fmt"
	"net/http"
)

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("%d %s", e.StatusCode, message)
}
