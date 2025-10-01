package yuna

import (
	"fmt"
	"net/http"

	"github.com/jkratz55/yuna/log"
)

func recovery() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				if err := recover(); err != nil && err != http.ErrAbortHandler {

					logger := log.LoggerFromCtx(r.Context())
					logger.Error(fmt.Sprintf("panic recovered: %v", err),
						log.PrettyStack())

					problem := InternalServerError()
					problem.ServeHTTP(w, r)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
