package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/fcjr/shiftapi"
)

type ChatMsg struct {
	Text string `json:"text"`
}

type EchoReply struct {
	Text string `json:"text"`
}

var echoWS = shiftapi.Websocket[struct{}](
	shiftapi.WSOn("chat", func(r *http.Request, ws *shiftapi.WSSender, _ struct{}, msg ChatMsg) error {
		return ws.Send(EchoReply{Text: "echo: " + msg.Text})
	}),
	shiftapi.WSSends(
		shiftapi.MessageType[EchoReply]("echo"),
	),
)

type Person struct {
	Name string `json:"name" validate:"required"`
}

type Greeting struct {
	Hello string `json:"hello"`
}

// WrongNameError is returned when the name is not "frank".
type WrongNameError struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func (e *WrongNameError) Error() string { return e.Message }

func greet(r *http.Request, in *Person) (*Greeting, error) {
	if in.Name != "frank" {
		return nil, &WrongNameError{
			Message: "wrong name, I only greet frank",
			Name:    in.Name,
		}
	}
	return &Greeting{Hello: in.Name}, nil
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

func search(r *http.Request, in SearchQuery) (*SearchResult, error) {
	return &SearchResult{
		Query: in.Q,
		Page:  in.Page,
		Limit: in.Limit,
	}, nil
}

type Status struct {
	OK bool `json:"ok"`
}

func health(r *http.Request, _ struct{}) (*Status, error) {
	return &Status{OK: true}, nil
}

type UploadInput struct {
	File *multipart.FileHeader `form:"file" validate:"required"`
}

type UploadResult struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

func upload(r *http.Request, in UploadInput) (*UploadResult, error) {
	return &UploadResult{
		Filename: in.File.Filename,
		Size:     in.File.Size,
	}, nil
}

type ImageUploadInput struct {
	Image *multipart.FileHeader `form:"image" accept:"image/png,image/jpeg" validate:"required"`
}

type ImageUploadResult struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

func uploadImage(r *http.Request, in ImageUploadInput) (*ImageUploadResult, error) {
	return &ImageUploadResult{
		Filename:    in.Image.Filename,
		ContentType: in.Image.Header.Get("Content-Type"),
		Size:        in.Image.Size,
	}, nil
}

type MultiUploadInput struct {
	Files []*multipart.FileHeader `form:"files" validate:"required"`
}

type MultiUploadResult struct {
	Count     int      `json:"count"`
	Filenames []string `json:"filenames"`
}

func uploadMulti(r *http.Request, in MultiUploadInput) (*MultiUploadResult, error) {
	names := make([]string, len(in.Files))
	for i, f := range in.Files {
		names[i] = fmt.Sprintf("%s (%d bytes)", f.Filename, f.Size)
	}
	return &MultiUploadResult{
		Count:     len(in.Files),
		Filenames: names,
	}, nil
}

func main() {
	api := shiftapi.New(shiftapi.WithInfo(shiftapi.Info{
		Title: "Greeter Demo API",
	}))

	shiftapi.Handle(api, "POST /greet", greet,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Greet a person",
			Description: "Greet a person with a friendly greeting",
			Tags:        []string{"greet"},
		}),
		shiftapi.WithError[*WrongNameError](http.StatusBadRequest),
	)

	shiftapi.Handle(api, "GET /search", search,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Search for things",
			Description: "Search with typed query parameters",
			Tags:        []string{"search"},
		}),
	)

	shiftapi.Handle(api, "GET /health", health,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary: "Health check",
			Tags:    []string{"health"},
		}),
	)

	shiftapi.Handle(api, "POST /upload", upload,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Upload a file",
			Description: "Upload a single file",
			Tags:        []string{"uploads"},
		}),
	)

	shiftapi.Handle(api, "POST /upload-image", uploadImage,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Upload an image",
			Description: "Upload a single image (PNG or JPEG only)",
			Tags:        []string{"uploads"},
		}),
	)

	shiftapi.Handle(api, "POST /upload-multi", uploadMulti,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Upload multiple files",
			Description: "Upload multiple files at once",
			Tags:        []string{"uploads"},
		}),
	)

	shiftapi.HandleWS(api, "GET /echo", echoWS,
		shiftapi.WithRouteInfo(shiftapi.RouteInfo{
			Summary:     "Echo WebSocket",
			Description: "Echoes back any message sent by the client",
			Tags:        []string{"websocket"},
		}),
	)

	log.Println("listening on :8080")
	log.Fatal(shiftapi.ListenAndServe(":8080", api))
	// docs at http://localhost:8080/docs
	// ws docs at http://localhost:8080/docs/ws
}
