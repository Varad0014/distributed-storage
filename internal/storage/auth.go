package storage

import (
	"crypto/subtle"
	"net/http"
)

func RequireStorageToken(expectedToken string, next http.Handler) http.Handler {
	// type cast
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			receivedToken := r.Header.Get("Authorization")
			expectedAuthorization := "Bearer " + expectedToken
			tokensMatch := subtle.ConstantTimeCompare(
				[]byte(receivedToken),
				[]byte(expectedAuthorization),
			)

			if tokensMatch != 1 {
				http.Error(
					w,
					"unauthorized",
					http.StatusUnauthorized,
				)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}
