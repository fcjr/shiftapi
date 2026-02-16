package server

import (
	"context"
	"net/http"

	"github.com/fcjr/shiftapi"
)

type EchoRequest struct {
	Message string `json:"message" validate:"required"`
}

type EchoResponse struct {
	Message string `json:"message"`
}

func echo(r *http.Request, body *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{Message: body.Message}, nil
}

type Status struct {
	OK bool `json:"ok"`
}

func health(r *http.Request) (*Status, error) {
	return &Status{OK: true}, nil
}

func ListenAndServe(ctx context.Context, addr string) error {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title: "{{name}}",
	}))

	shiftapi.Post(api, "/echo", echo,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Echo a message",
			Description: "Returns the message you send",
			Tags:        []string{"echo"},
		}),
	)

	shiftapi.Get(api, "/health", health,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary: "Health check",
			Tags:    []string{"health"},
		}),
	)

	return shiftapi.ListenAndServe(addr, api)
}
