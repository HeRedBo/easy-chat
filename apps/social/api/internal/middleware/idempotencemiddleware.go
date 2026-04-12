package middleware

import (
	"net/http"

	"github.com/HeRedBo/easy-chat/pkg/interceptor"
)

type IdempotenceMiddleware struct {
}

func NewIdempotenceMiddleware() *IdempotenceMiddleware {
	return &IdempotenceMiddleware{}
}

func (m *IdempotenceMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO generate middleware implement function, delete after code implementation
		r = r.WithContext(interceptor.ContextWithIdempotentID(r.Context()))
		// Passthrough to next handler if need
		next(w, r)
	}
}
