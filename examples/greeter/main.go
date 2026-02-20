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

type SearchQuery struct {
	Q     string `query:"q"     validate:"required"`
	Page  int    `query:"page"  validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=100"`
}

type SearchResult struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

func search(r *http.Request, query SearchQuery) (*SearchResult, error) {
	return &SearchResult{
		Query: query.Q,
		Page:  query.Page,
		Limit: query.Limit,
	}, nil
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

	shiftapi.GetWithQuery(api, "/search", search,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Search for things",
			Description: "Search with typed query parameters",
			Tags:        []string{"search"},
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
