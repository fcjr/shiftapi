
<p align="center">
<img src="assets/logo.svg" alt="ShiftAPI Logo">
</p>

# ShiftAPI

Quickly write RESTful APIs in go with automatic openapi schema generation.

<!-- [![GitHub release (latest by date)][release-img]][release] -->
[![GolangCI][golangci-lint-img]][golangci-lint]
[![Go Report Card][report-card-img]][report-card]

## NOTE: THIS IS AN EXPERIMENT

This project is highly experimental -- the API is likely to change (currently only _basic_ post requests are even implemented).
This is **in no way production ready**.

This project was inspired by the simplicity of [FastAPI](https://github.com/tiangolo/fastapi).

Due to limitations of typing in go this library will probably not be production ready pre go 1.18 as handlers must be passed as interfaces{} and validated at _runtime_ (Scary I know! 😱).  Once generics hit I hope to rewrite the handler implementation to restore compile time type checking & safety.

## Installation

```sh
go get github.com/fcjr/shiftapi
```

## Usage

```go
package main

import (
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

// This is your http handler!
// ShiftAPI is responsible for marshalling the request body and marshalling the return value.
func greeter(p *Person) (*Greeting, *shiftapi.Error) {
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
		Description: "It greets you by name.",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(api.Serve())
	// redoc will be served at http://localhost:8080/docs
}
```

[release-img]: https://img.shields.io/github/v/release/fcjr/shiftapi
[release]: https://github.com/fcjr/shiftapi/releases
[golangci-lint-img]: https://github.com/fcjr/shiftapi/workflows/go-lint/badge.svg
[golangci-lint]: https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint
[report-card-img]: https://goreportcard.com/badge/github.com/fcjr/shiftapi
[report-card]: https://goreportcard.com/report/github.com/fcjr/shiftapi
