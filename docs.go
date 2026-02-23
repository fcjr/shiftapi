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
<script src="https://cdnjs.cloudflare.com/ajax/libs/scalar-api-reference/1.36.2/standalone.min.js" integrity="sha512-1eGM3+sAmNpB7cn/i3KOVszLEAph0LC96/Qk1T0hf/eK8p0MSU7og2mx0P0bv5R4R8U7LWJnA9cDCxp7RRdF/Q==" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
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
