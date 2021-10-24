package main

import (
	"encoding/json"
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

func BadRequestError(code, msg string) *shiftapi.Error {
	type err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	b, _ := json.Marshal(&err{
		Code:    code,
		Message: msg,
	})
	return &shiftapi.Error{
		Code: http.StatusBadRequest,
		Body: b,
	}
}

func greeter(p *Person) (*Greeting, *shiftapi.Error) {
	if p.Name != "frank" {
		return nil, BadRequestError("wrong_name", "I only greet frank.")
	}
	return &Greeting{
		Hello: p.Name,
	}, nil
}

func main() {

	api := shiftapi.New(&shiftapi.Params{
		SchemaInfo: &shiftapi.SchemaParams{
			Title: "Greeter Demo API",
		},
	})

	err := api.POST("/greet", greeter, http.StatusOK, &shiftapi.HandlerOpts{
		Summary:     "Greeter Method",
		Description: "It greets anyone named 'frank'",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(api.Serve())
	// redoc will be served at http://localhost:8080/docs
}
