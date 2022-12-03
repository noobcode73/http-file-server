package server

import "html/template"

const directoryListingTemplateText = `
<html>
<head>
	<title>{{ .Title }}</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>body{font-family: sans-serif;width: 90%;padding-left: 5%;padding-top: 10px;}td{padding:.5em;}a{display:block;}tbody tr:nth-child(odd){background:#eee;}.number{text-align:right}.text{text-align:left;word-break:break-all;}canvas,table{width:100%;max-width:100%;}</style>
</head>
<body>
<h1>{{ .Title }}</h1>
{{ if or .Files .AllowUpload }}
<div>
<a href="{{ .TarGzURL }}">.tar.gz of all files</a>
<a href="{{ .ZipURL }}">.zip of all files</a>
</div>
<br>
<div>
{{ if .AllowCreate }}
	<input type="text" placeholder="Name new folder" id="newfolder">
	<button type="button" id="btn_newFolder" onclick="create()">Create</button>
{{- end }}
</div>
<hr>
<table>
	<thead>
		<th>Name</th>
		<th>Modified</th>
		<th>Type</th>
		<th class=number>Size (bytes)</th>
	</thead>
	<tbody>
	<tr><td colspan=4><a href="../">..</a></td></tr>
	{{- range .Files }}
	<tr>
		<td class=text><a href="{{ .URL.String }}">{{ .Name }}</td>
		<td>{{ .Modified }}</td>
		{{ if (not .IsDir) }}
		<td>{{ .Type }}</td>
		<td class=number>{{ .Size.String }} ({{ .Size | printf "%d" }})</td>
		{{ else }}
		<td>{{ .Type }} [files in: {{ .FCount }}]</td>
		<td class=number>---</td>
		{{ end }}
	</tr>
	{{- end }}
	{{- if .AllowUpload }}
	<tr><td colspan=4><form method="post" enctype="multipart/form-data"><input required name="file" type="file multiple"/><input value="Upload" type="submit"/></form></td></tr>
	{{- end }}
	</tbody>
</table>
{{ end }}
<script type="text/javascript">

 function create() {
        const name = document.getElementById("newfolder").value
        if (name.length === 0)
            return

        send(window.location.href + "?new", {
            method: 'POST',
            headers: {"Content-Type": "application/x-www-form-urlencoded"},
            body: "name=" + name
        })
    }

    function send(url, options) {
        fetch(url, options).then((response) => {
            if (!response.ok) {
                alert("HTTP error: "+ response.statusText + "! Status: " + response.status);
            } else {
                alert("Success")
                window.location.reload();
            }
        }).catch(err => {
            alert(err)
        });
    }

</script>
</body>
</html>
`

var (
	directoryListingTemplate = template.Must(template.New("").Parse(directoryListingTemplateText))
)
