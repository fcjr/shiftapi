package shiftapi_test

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/fcjr/shiftapi"
)

func Example() {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title:   "My API",
		Version: "1.0.0",
	}))

	type HelloRequest struct {
		Name string `json:"name" validate:"required"`
	}
	type HelloResponse struct {
		Message string `json:"message"`
	}

	shiftapi.Post(api, "/hello", func(r *http.Request, in HelloRequest) (*HelloResponse, error) {
		return &HelloResponse{Message: "Hello, " + in.Name + "!"}, nil
	})

	log.Fatal(shiftapi.ListenAndServe(":8080", api))
}

func ExampleNew() {
	api := shiftapi.New(
		shiftapi.WithInfo(shiftapi.Info{
			Title:       "Pet Store",
			Version:     "2.0.0",
			Description: "A sample pet store API",
		}),
		shiftapi.WithMaxUploadSize(10<<20), // 10 MB
	)
	_ = api
}

func ExampleGet() {
	api := shiftapi.New()

	type UserQuery struct {
		ID int `query:"id" validate:"required"`
	}
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	shiftapi.Get(api, "/user", func(r *http.Request, in UserQuery) (*User, error) {
		return &User{ID: in.ID, Name: "Alice"}, nil
	})
}

func ExampleGet_pathParameter() {
	api := shiftapi.New()

	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	shiftapi.Get(api, "/users/{id}", func(r *http.Request, _ struct{}) (*User, error) {
		id := r.PathValue("id")
		return &User{ID: id, Name: "Alice"}, nil
	})

	// Make a request to verify.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/42", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output:
	// {"id":"42","name":"Alice"}
}

func ExamplePost() {
	api := shiftapi.New()

	type CreateInput struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}
	type CreateOutput struct {
		ID int `json:"id"`
	}

	shiftapi.Post(api, "/users", func(r *http.Request, in CreateInput) (*CreateOutput, error) {
		return &CreateOutput{ID: 1}, nil
	}, shiftapi.WithStatus(http.StatusCreated))
}

func ExamplePost_queryAndBody() {
	api := shiftapi.New()

	type Request struct {
		Version string `query:"v"`
		Name    string `json:"name"`
	}
	type Response struct {
		Result string `json:"result"`
	}

	shiftapi.Post(api, "/action", func(r *http.Request, in Request) (*Response, error) {
		return &Response{Result: in.Name + " (v" + in.Version + ")"}, nil
	})
	_ = api
}

func ExamplePost_fileUpload() {
	api := shiftapi.New()

	type UploadInput struct {
		File *multipart.FileHeader `form:"file" validate:"required"`
	}
	type UploadResult struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	shiftapi.Post(api, "/upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return &UploadResult{
			Filename: in.File.Filename,
			Size:     in.File.Size,
		}, nil
	})
}

func ExampleError() {
	api := shiftapi.New()

	type Empty struct{}

	shiftapi.Get(api, "/secret", func(r *http.Request, _ struct{}) (*Empty, error) {
		token := r.Header.Get("Authorization")
		if token == "" {
			return nil, shiftapi.Error(http.StatusUnauthorized, "missing auth token")
		}
		return &Empty{}, nil
	})

	// Make a request without auth to verify.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/secret", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 401
	// {"message":"missing auth token"}
}

func ExampleWithRouteInfo() {
	api := shiftapi.New()

	shiftapi.Get(api, "/health", func(r *http.Request, _ struct{}) (*struct {
		OK bool `json:"ok"`
	}, error) {
		return &struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	}, shiftapi.WithRouteInfo(shiftapi.RouteInfo{
		Summary:     "Health check",
		Description: "Returns the health status of the service.",
		Tags:        []string{"monitoring"},
	}))
}

func ExampleWithStatus() {
	api := shiftapi.New()

	type Item struct {
		Name string `json:"name"`
	}
	type Created struct {
		ID int `json:"id"`
	}

	shiftapi.Post(api, "/items", func(r *http.Request, in Item) (*Created, error) {
		return &Created{ID: 1}, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	_ = api
}

func ExampleAPI_ServeHTTP() {
	api := shiftapi.New()

	shiftapi.Get(api, "/ping", func(r *http.Request, _ struct{}) (*struct {
		Pong bool `json:"pong"`
	}, error) {
		return &struct {
			Pong bool `json:"pong"`
		}{Pong: true}, nil
	})

	// Use as http.Handler in tests.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ping", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output:
	// {"pong":true}
}
