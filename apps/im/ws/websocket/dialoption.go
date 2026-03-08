package websocket

import "net/http"

type DialOptions func(option *dialOption)

type dialOption struct {
	pattern string
	header  http.Header
}

func newDailOptions(opts ...DialOptions) dialOption {
	o := dialOption{
		pattern: "/ws",
		header:  nil,
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}
func WithClientPatten(pattern string) DialOptions {
	return func(opt *dialOption) {
		opt.pattern = pattern
	}
}
func WithClientHeader(header http.Header) DialOptions {
	return func(opt *dialOption) {
		opt.header = header
	}
}
