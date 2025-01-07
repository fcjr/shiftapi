
<p align="center">
	<img src="assets/logo.svg" alt="ShiftAPI Logo">
</p>

# ShiftAPI

Quickly write RESTful APIs in go with automatic openapi schema generation.

Inspired by the simplicity of [FastAPI](https://github.com/tiangolo/fastapi).

<!-- [![GitHub release (latest by date)][release-img]][release] -->
[![GolangCI][golangci-lint-img]][golangci-lint]
[![Go Report Card][report-card-img]][report-card]

## Installation

```sh
go get github.com/fcjr/shiftapi
```

## Usage

```go
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
        return &Greeting{
            Hello: person.Name,
        }, nil
    }

    func main() {
        ctx := context.Background()
        server := shiftapi.New(ctx, shiftapi.WithInfo(shiftapi.Info{
            Title: "Geeter Demo API",
            Description: "It greets you by name.",
        }))

        handleGreet := shiftapi.Post("/greet", greet)
        _ = server.Register(handleGreet) // You should handle errors in production code.

        log.Fatal(server.ListenAndServe(":8080"))
        // redoc will be served at http://localhost:8080/docs
    }
```

[release-img]: https://img.shields.io/github/v/release/fcjr/shiftapi
[release]: https://github.com/fcjr/shiftapi/releases
[golangci-lint-img]: https://github.com/fcjr/shiftapi/workflows/go-lint/badge.svg
[golangci-lint]: https://github.com/fcjr/shiftapi/actions?query=workflow%3Ago-lint
[report-card-img]: https://goreportcard.com/badge/github.com/fcjr/shiftapi
[report-card]: https://goreportcard.com/report/github.com/fcjr/shiftapi
