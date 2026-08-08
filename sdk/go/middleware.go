package wt

import "net/http"

// recoverHTTP wraps an HTTP handler so panics are reported and converted into
// a 500 response. The panic is not re-raised, keeping the server alive.
func (c *Client) recoverHTTP(w http.ResponseWriter, r *http.Request, next http.Handler, opts []ReportOption) {
	defer func() {
		if v := recover(); v != nil {
			c.reportPanic(v, append(opts, WithContext(map[string]any{
				"url":    r.URL.Path,
				"method": r.Method,
			}))...)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()
	next.ServeHTTP(w, r)
}

// RecoverMiddleware wraps an http.Handler so panics are reported to WatchTower
// and converted into a 500 response. Use it around your router.
func (c *Client) RecoverMiddleware(opts ...ReportOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.recoverHTTP(w, r, next, opts)
		})
	}
}

// RecoverMiddleware wraps an http.Handler using the process-wide client (see
// Init). Panics are reported and converted into a 500 response.
func RecoverMiddleware(opts ...ReportOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c := defaultClient(); c != nil {
				c.recoverHTTP(w, r, next, opts)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// recoverFn wraps fn (a goroutine body or NATS handler) so panics are reported
// and then re-raised, preserving the caller's error-handling contract.
func (c *Client) recoverFn(fn func(), opts []ReportOption) {
	defer func() {
		if v := recover(); v != nil {
			c.reportPanic(v, opts...)
			panic(v)
		}
	}()
	fn()
}

// RecoverHandler wraps fn so panics are reported to WatchTower and re-raised.
func (c *Client) RecoverHandler(fn func(), opts ...ReportOption) func() {
	return func() { c.recoverFn(fn, opts) }
}

// RecoverHandler wraps fn using the process-wide client (see Init). Panics are
// reported and re-raised.
func RecoverHandler(fn func(), opts ...ReportOption) func() {
	return func() {
		if c := defaultClient(); c != nil {
			c.recoverFn(fn, opts)
			return
		}
		fn()
	}
}
