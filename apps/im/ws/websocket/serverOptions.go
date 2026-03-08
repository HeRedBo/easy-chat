package websocket

type ServerOptions func(opt *serverOption)

type serverOption struct {
	Authentication
	patten string
}

func NewServerOption(opts ...ServerOptions) serverOption {

	o := serverOption{
		Authentication: new(authentication),
		patten:         "/ws",
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func WithServerAuthentication(auth Authentication) ServerOptions {
	return func(opt *serverOption) {
		opt.Authentication = auth
	}
}

func WithServerPatten(patten string) ServerOptions {
	return func(opt *serverOption) {
		opt.patten = patten
	}
}
