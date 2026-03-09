package shiftapi_test

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/fcjr/shiftapi"
)

type exampleNotFoundError struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (e *exampleNotFoundError) Error() string { return e.Message }

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

	shiftapi.Handle(api, "POST /hello", func(r *http.Request, in HelloRequest) (*HelloResponse, error) {
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

func ExampleHandle_get() {
	api := shiftapi.New()

	type UserQuery struct {
		ID int `query:"id" validate:"required"`
	}
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	shiftapi.Handle(api, "GET /user", func(r *http.Request, in UserQuery) (*User, error) {
		return &User{ID: in.ID, Name: "Alice"}, nil
	})
}

func ExampleHandle_pathParameter() {
	api := shiftapi.New()

	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	shiftapi.Handle(api, "GET /users/{id}", func(r *http.Request, _ struct{}) (*User, error) {
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

func ExampleHandle_post() {
	api := shiftapi.New()

	type CreateInput struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}
	type CreateOutput struct {
		ID int `json:"id"`
	}

	shiftapi.Handle(api, "POST /users", func(r *http.Request, in CreateInput) (*CreateOutput, error) {
		return &CreateOutput{ID: 1}, nil
	}, shiftapi.WithStatus(http.StatusCreated))
}

func ExampleHandle_queryAndBody() {
	api := shiftapi.New()

	type Request struct {
		Version string `query:"v"`
		Name    string `json:"name"`
	}
	type Response struct {
		Result string `json:"result"`
	}

	shiftapi.Handle(api, "POST /action", func(r *http.Request, in Request) (*Response, error) {
		return &Response{Result: in.Name + " (v" + in.Version + ")"}, nil
	})
	_ = api
}

func ExampleHandle_fileUpload() {
	api := shiftapi.New()

	type UploadInput struct {
		File *multipart.FileHeader `form:"file" validate:"required"`
	}
	type UploadResult struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	shiftapi.Handle(api, "POST /upload", func(r *http.Request, in UploadInput) (*UploadResult, error) {
		return &UploadResult{
			Filename: in.File.Filename,
			Size:     in.File.Size,
		}, nil
	})
}

type exampleAuthError struct {
	Message string `json:"message"`
}

func (e *exampleAuthError) Error() string { return e.Message }

func ExampleWithError_auth() {
	api := shiftapi.New()

	type Empty struct{}

	shiftapi.Handle(api, "GET /secret", func(r *http.Request, _ struct{}) (*Empty, error) {
		token := r.Header.Get("Authorization")
		if token == "" {
			return nil, &exampleAuthError{Message: "missing auth token"}
		}
		return &Empty{}, nil
	}, shiftapi.WithError[*exampleAuthError](http.StatusUnauthorized))

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

func ExampleWithError() {
	api := shiftapi.New()

	shiftapi.Handle(api, "GET /users/{id}", func(r *http.Request, _ struct{}) (*struct {
		Name string `json:"name"`
	}, error) {
		return nil, &exampleNotFoundError{Message: "user not found", Detail: "no user with that ID"}
	}, shiftapi.WithError[*exampleNotFoundError](http.StatusNotFound))

	// Make a request to verify.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/42", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 404
	// {"message":"user not found","detail":"no user with that ID"}
}

func ExampleWithRouteInfo() {
	api := shiftapi.New()

	shiftapi.Handle(api, "GET /health", func(r *http.Request, _ struct{}) (*struct {
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

	shiftapi.Handle(api, "POST /items", func(r *http.Request, in Item) (*Created, error) {
		return &Created{ID: 1}, nil
	}, shiftapi.WithStatus(http.StatusCreated))

	_ = api
}

func ExampleAPI_Group() {
	api := shiftapi.New()

	v1 := api.Group("/api/v1")

	shiftapi.Handle(v1, "GET /users", func(r *http.Request, _ struct{}) (*struct {
		Name string `json:"name"`
	}, error) {
		return &struct {
			Name string `json:"name"`
		}{Name: "Alice"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output:
	// {"name":"Alice"}
}

func ExampleHandle_responseHeaders() {
	api := shiftapi.New()

	type CachedItem struct {
		CacheControl string  `header:"Cache-Control"`
		ETag         *string `header:"ETag"`
		Name         string  `json:"name"`
	}

	shiftapi.Handle(api, "GET /item", func(r *http.Request, _ struct{}) (*CachedItem, error) {
		etag := `"v1"`
		return &CachedItem{
			CacheControl: "max-age=3600",
			ETag:         &etag,
			Name:         "Widget",
		}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/item", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Header().Get("Cache-Control"))
	fmt.Println(w.Header().Get("ETag"))
	fmt.Println(w.Body.String())
	// Output:
	// max-age=3600
	// "v1"
	// {"name":"Widget"}
}

func ExampleWithResponseHeader() {
	api := shiftapi.New(
		shiftapi.WithResponseHeader("X-Content-Type-Options", "nosniff"),
	)

	shiftapi.Handle(api, "GET /item", func(r *http.Request, _ struct{}) (*struct {
		Name string `json:"name"`
	}, error) {
		return &struct {
			Name string `json:"name"`
		}{Name: "Widget"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/item", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Header().Get("X-Content-Type-Options"))
	fmt.Println(w.Body.String())
	// Output:
	// nosniff
	// {"name":"Widget"}
}

func ExampleHandle_noContent() {
	api := shiftapi.New()

	shiftapi.Handle(api, "DELETE /items/{id}", func(r *http.Request, _ struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, shiftapi.WithStatus(http.StatusNoContent))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/items/42", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Code)
	fmt.Println(w.Body.String())
	// Output:
	// 204
	//
}

func ExampleFromContext() {
	userKey := shiftapi.NewContextKey[string]("user")

	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, shiftapi.SetContext(r, userKey, "alice"))
		})
	}

	api := shiftapi.New(shiftapi.WithMiddleware(authMiddleware))

	shiftapi.Handle(api, "GET /whoami", func(r *http.Request, _ struct{}) (*struct {
		User string `json:"user"`
	}, error) {
		user, _ := shiftapi.FromContext(r, userKey)
		return &struct {
			User string `json:"user"`
		}{User: user}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/whoami", nil)
	api.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output:
	// {"user":"alice"}
}

func ExampleAPI_ServeHTTP() {
	api := shiftapi.New()

	shiftapi.Handle(api, "GET /ping", func(r *http.Request, _ struct{}) (*struct {
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
