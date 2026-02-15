package shiftapi

import (
	"html/template"
	"io"
)

const docsTemplate string = `<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
<div id="app"></div>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
<script>
Scalar.createApiReference('#app', {
  url: '{{.SpecURL}}',
})
</script>
</body>
</html>
`

type docsData struct {
	Title   string
	SpecURL string
}

func genDocsHTML(data docsData, out io.Writer) error {
	t, err := template.New("docsHTML").Parse(docsTemplate)
	if err != nil {
		return err
	}
	return t.Execute(out, data)
}
