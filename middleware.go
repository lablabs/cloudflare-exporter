package main

import "net/http"

type HeaderMiddleware struct {
	key   string
	value string
	next  http.RoundTripper
}

func NewHeaderMiddleware(key, value string, next http.RoundTripper) *HeaderMiddleware {
	if next == nil {
		next = http.DefaultTransport
	}

	return &HeaderMiddleware{
		key:   key,
		value: value,
		next:  next,
	}
}

func (m *HeaderMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(m.key, m.value)
	return m.next.RoundTrip(req)
}
