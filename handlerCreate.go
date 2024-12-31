package shiftapi

import "net/http"

func Get[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodGet,
		path:        path,
		handlerFunc: handlerFunc,
		options:     options,
	}
}

func Post[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodPost,
		path:        path,
		handlerFunc: handlerFunc,
		options:     options,
	}
}

func Put[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodPut,
		path:        path,
		handlerFunc: handlerFunc,
	}
}
func Patch[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodPatch,
		path:        path,
		handlerFunc: handlerFunc,
	}
}

func Delete[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodDelete,
		path:        path,
		handlerFunc: handlerFunc,
	}
}

func Head[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodHead,
		path:        path,
		handlerFunc: handlerFunc,
	}
}

func Options[RequestBody any, ResponseBody any](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(HandlerFunc[RequestBody, ResponseBody]) HandlerFunc[RequestBody, ResponseBody],
) Handler {
	return &handler[RequestBody, ResponseBody]{
		method:      http.MethodOptions,
		path:        path,
		handlerFunc: handlerFunc,
	}
}
