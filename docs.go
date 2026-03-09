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

const asyncDocsTemplate string = `<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://unpkg.com/@asyncapi/react-component@3.0.2/styles/default.min.css" integrity="sha384-+kAXZlmkYbACsvDm+h2/qAphvw98RHOGObISB6ouInRvC2tvmBLwvgZVZQOtMndl" crossorigin="anonymous">
</head>
<body>
<div id="asyncapi"></div>
<script src="https://unpkg.com/@asyncapi/react-component@3.0.2/browser/standalone/index.js" integrity="sha384-qYnchRkiLeA3INQMui0zmEqOZzAdSM6DTME5EPknhPDJNfi5FkyRVoSKfswOT1K/" crossorigin="anonymous"></script>
<script>
AsyncApiStandalone.render({
  schema: { url: '{{.SpecURL}}' },
  config: { show: { sidebar: true } },
}, document.getElementById('asyncapi'));
</script>
</body>
</html>
`

var (
	docsTmpl      = template.Must(template.New("docsHTML").Parse(docsTemplate))
	asyncDocsTmpl = template.Must(template.New("asyncDocsHTML").Parse(asyncDocsTemplate))
)

type docsData struct {
	Title   string
	SpecURL string
}

func genDocsHTML(data docsData, out io.Writer) error {
	return docsTmpl.Execute(out, data)
}

func genAsyncDocsHTML(data docsData, out io.Writer) error {
	return asyncDocsTmpl.Execute(out, data)
}
