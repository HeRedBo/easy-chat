package websocket

import (
	"fmt"
	"net/http"
	"time"
)

type Authentication interface {
	Auth(w http.ResponseWriter, r *http.Request) bool
	UserId(r *http.Request) string
}

type authentication struct{}

func (*authentication) Auth(w http.ResponseWriter, r *http.Request) bool {
	return true
}

func (*authentication) UserId(r *http.Request) string {
	query := r.URL.Query()
	if query != nil && query["user_id"] != nil {
		return fmt.Sprintf("%v", query.Get("user_id"))
	}
	return fmt.Sprintf("%v", time.Now().UnixMilli())
}
