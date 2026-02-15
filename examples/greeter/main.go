package main

import (
	"log"
	"net/http"

	"github.com/fcjr/shiftapi"
)

type Person struct {
	Name string `json:"name" validate:"required"`
}

type Greeting struct {
	Hello string `json:"hello"`
}

func greet(r *http.Request, body *Person) (*Greeting, error) {
	if body.Name != "frank" {
		return nil, shiftapi.Error(http.StatusBadRequest, "wrong name, I only greet frank")
	}
	return &Greeting{Hello: body.Name}, nil
}

type Status struct {
	OK bool `json:"ok"`
}

func health(r *http.Request) (*Status, error) {
	return &Status{OK: true}, nil
}

func main() {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title: "Greeter Demo API",
	}))

	shiftapi.Post(api, "/greet", greet,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Greet a person",
			Description: "Greet a person with a friendly greeting",
			Tags:        []string{"greet"},
		}),
	)

	shiftapi.Get(api, "/health", health,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary: "Health check",
			Tags:    []string{"health"},
		}),
	)

	log.Println("listening on :8080")
	log.Fatal(shiftapi.ListenAndServe(":8080", api))
	// docs at http://localhost:8080/docs
}
