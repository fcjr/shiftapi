package shiftapi

type HandlerInfo struct {
	Summary     string
	Description string
	Tags        []string
}

type optionsApplier interface {
	setInfo(*HandlerInfo)
}

func (h *handler[RequestBody, ResponseBody]) setInfo(info *HandlerInfo) {
	h.info = info
}

var _ optionsApplier = (*handler[ValidBody, ValidBody])(nil)

func WithHandlerInfo(info *HandlerInfo) func(Handler) Handler {
	return func(h Handler) Handler {
		if handler, ok := h.(optionsApplier); ok {
			handler.setInfo(info)
			return h
		}
		panic("invalid handler type, this should never happen, please open an issue on github")
	}
}
