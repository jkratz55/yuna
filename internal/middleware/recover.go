package middleware

import (
	"net/http"

	"github.com/jkratz55/yuna/log"
)

// todo: implement middleware to recover from panic

func Recover(logger *log.Logger) func(next http.Handler) http.Handler {
	return nil
}
