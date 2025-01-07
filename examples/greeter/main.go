package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/fcjr/shiftapi"
)

type Person struct {
	Name string `json:"name"`
}

type Greeting struct {
	Hello string `json:"hello"`
}

func greet(ctx context.Context, headers http.Header, person *Person) (*Greeting, error) {
	if person.Name != "frank" {
		return nil, errors.New("wrong name, I only greet frank")
	}
	return &Greeting{
		Hello: person.Name,
	}, nil
}

func main() {
	ctx := context.Background()
	server := shiftapi.New(ctx, shiftapi.WithServerInfo(shiftapi.ServerInfo{
		Title: "Geeter Demo API",
	}))

	handleGreet := shiftapi.Post(
		"/greet",
		greet,
		shiftapi.WithHandlerInfo(&shiftapi.HandlerInfo{
			Summary:     "Greet a person",
			Description: "Greet a person with a friendly greeting",
			Tags:        []string{"greet"},
		}),
	)
	if err := server.Register(handleGreet); err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.ListenAndServe(":8080"))
	// redoc will be served at http://localhost:8080/docs
}
