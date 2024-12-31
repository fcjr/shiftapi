package shiftapi

import (
	"html/template"
	"io"
)

const redocTemplate string = `<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="shortcut icon" href="{{.FaviconURL}}">
<style>
  body {
    margin: 0;
    padding: 0;
  }
</style>
</head>
<body>
<redoc spec-url="{{.SpecURL}}"></redoc>
<script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"> </script>
</body>
</html>
`

type redocData struct {
	Title      string
	FaviconURL string
	RedocURL   string
	SpecURL    string
	// EnableGoogleFonts bool
}

func genRedocHTML(data redocData, out io.Writer) error {
	t, err := template.New("redocHTML").Parse(redocTemplate)
	if err != nil {
		return err
	}

	err = t.Execute(out, data)
	if err != nil {
		return err
	}

	return nil
}
