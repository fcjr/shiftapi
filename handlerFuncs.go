package shiftapi

import "net/http"

func method[RequestBody ValidBody, ResponseBody ValidBody](
	method string,
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	h := &handler[RequestBody, ResponseBody]{
		method:      method,
		path:        path,
		handlerFunc: handlerFunc,
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func Get[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodGet, path, handlerFunc, options...)
}

func Post[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodPost, path, handlerFunc, options...)
}

func Put[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodPut, path, handlerFunc, options...)
}
func Patch[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodPatch, path, handlerFunc, options...)
}

func Delete[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodDelete, path, handlerFunc, options...)
}

func Head[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodHead, path, handlerFunc, options...)
}

func Options[RequestBody ValidBody, ResponseBody ValidBody](
	path string,
	handlerFunc HandlerFunc[RequestBody, ResponseBody],
	options ...func(Handler) Handler,
) *handler[RequestBody, ResponseBody] {
	return method(http.MethodOptions, path, handlerFunc, options...)
}
