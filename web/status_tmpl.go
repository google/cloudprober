package web

var statusTmpl = `
<html>

<style type="text/css">
table.status-list {
  border-collapse: collapse;
  border-spacing: 0;
	margin-bottom: 40px;
	font-family: monospace;
}
table.status-list td,th {
  border: 1px solid gray;
  padding: 0.25em 0.5em;
	max-width: 200px;
}
pre {
    white-space: pre-wrap;
		word-wrap: break-word;
}
</style>

<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
`
