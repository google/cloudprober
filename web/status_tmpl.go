package web

var statusTmpl = `
<html>

<head>
<style type="text/css">
body {
  font-family: "Roboto","Helvetica","Arial",sans-serif;
	font-size: 14px;
}

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
</head>

<b>Started</b>: {{.StartTime}} -- up {{.Uptime}}<br/>
<b>Version</b>: {{.Version}}<br>
<b>Config</b>: <a href="/config">/config</a><br>

<h3>Probes:</h3>
{{.ProbesStatus}}

<h3>Surfacers:</h3>
{{.SurfacersStatus}}

<h3>Servers:</h3>
{{.ServersStatus}}
`
