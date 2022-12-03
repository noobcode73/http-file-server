package main

import (
	"github.com/dastoori/higgs"
	"io"
	"path/filepath"
	"strconv"
)

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	tarGzKey         = "tar.gz"
	tarGzValue       = "true"
	tarGzContentType = "application/x-tar+gzip"
	zipKey           = "zip"
	zipValue         = "true"
	zipContentType   = "application/zip"
	osPathSeparator  = string(filepath.Separator)
)

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

        send(window.location.href, {
            method: 'PUT',
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

func isHidden(p string) bool {
	h, err := higgs.IsHidden(p)
	if err != nil {
		fmt.Println(err)
	}
	return h
}

type fileSizeBytes int64

func (f fileSizeBytes) String() string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	divBy := func(x int64) int {
		return int(math.Round(float64(f) / float64(x)))
	}
	switch {
	case f < KB:
		return fmt.Sprintf("%db", f)
	case f < MB:
		return fmt.Sprintf("%dkb", divBy(KB))
	case f < GB:
		return fmt.Sprintf("%dmb", divBy(MB))
	case f >= GB:
		fallthrough
	default:
		return fmt.Sprintf("%dgb", divBy(GB))
	}
}

type directoryListingFileData struct {
	Name     string
	Size     fileSizeBytes
	IsDir    bool
	Type     string
	FCount   int
	Modified string
	URL      *url.URL
	IsHidden bool
}

type directoryListingData struct {
	Title         string
	ZipURL        *url.URL
	TarGzURL      *url.URL
	Files         []directoryListingFileData
	AllowUpload   bool
	AllowDelete   bool
	AllowCreate   bool
	NoAllowHidden bool
}

type fileHandler struct {
	route          string
	path           string
	allowUpload    bool
	allowDelete    bool
	allowCreate    bool
	customTemplate string
	noAllowHidden  bool
}

var (
	directoryListingTemplate = template.Must(template.New("").Parse(directoryListingTemplateText))
)

func (f *fileHandler) serveStatus(w http.ResponseWriter, r *http.Request, status int) error {
	w.WriteHeader(status)
	responseText := []byte(http.StatusText(status))
	if f.customTemplate != "" {
		p := f.customTemplate + osPathSeparator + "errors" + osPathSeparator + strconv.Itoa(status) + ".html"
		responseText, _ = os.ReadFile(p)
	}
	_, err := w.Write(responseText)
	if err != nil {
		return err
	}
	return nil
}

func (f *fileHandler) serveTarGz(w http.ResponseWriter, r *http.Request, path string) error {
	w.Header().Set("Content-Type", tarGzContentType)
	name := filepath.Base(path) + ".tar.gz"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return tarGz(w, path)
}

func (f *fileHandler) serveZip(w http.ResponseWriter, r *http.Request, osPath string) error {
	w.Header().Set("Content-Type", zipContentType)
	name := filepath.Base(osPath) + ".zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return zip(w, osPath)
}

func (f *fileHandler) serveDir(w http.ResponseWriter, r *http.Request, osPath string) error {
	d, err := os.Open(osPath)
	if err != nil {
		return err
	}
	files, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name()) })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if f.customTemplate != "" {
		var e error
		directoryListingTemplate, e = template.ParseFiles(f.customTemplate + osPathSeparator + "base.html")
		if e != nil {
			fmt.Println("can`t load custom template", e)
		}
	}

	return directoryListingTemplate.Execute(w, directoryListingData{
		AllowUpload:   f.allowUpload,
		AllowDelete:   f.allowDelete,
		AllowCreate:   f.allowCreate,
		NoAllowHidden: f.noAllowHidden,
		Title: func() string {
			relPath, _ := filepath.Rel(f.path, osPath)
			//return filepath.Join(filepath.Base(f.path), relPath)
			tPath := path.Join(filepath.Base(f.path), relPath)
			return strings.Replace(tPath, "\\", "/", -1)
		}(),
		TarGzURL: func() *url.URL {
			u := *r.URL
			q := u.Query()
			q.Set(tarGzKey, tarGzValue)
			u.RawQuery = q.Encode()
			return &u
		}(),
		ZipURL: func() *url.URL {
			u := *r.URL
			q := u.Query()
			q.Set(zipKey, zipValue)
			u.RawQuery = q.Encode()
			return &u
		}(),
		Files: func() (out []directoryListingFileData) {
			// first directories then files (sorted)
			var filesList []directoryListingFileData
			for _, d := range files {
				name := d.Name()
				absPath := osPath + osPathSeparator + name
				hidden := isHidden(absPath)
				if f.noAllowHidden && hidden {
					continue
				}

				fType := "DIR"
				fCount := 0
				if d.IsDir() {
					subD, e := os.ReadDir(absPath + osPathSeparator)
					if e == nil {
						fCount = len(subD)
					} else {
						fmt.Println(e)
						fmt.Println(absPath)
					}
				} else {
					fType = strings.Replace(filepath.Ext(name), ".", "", 1)
					if fType == "" {
						fType = "File"
					}
				}
				fileData := directoryListingFileData{
					Name:     name,
					IsDir:    d.IsDir(),
					Size:     fileSizeBytes(d.Size()),
					Type:     fType,
					FCount:   fCount,
					IsHidden: hidden,
					Modified: d.ModTime().Format("2006-01-02 15:04:05"),
					URL: func() *url.URL {
						u := *r.URL
						u.Path = path.Join(u.Path, name)
						if d.IsDir() {
							u.Path += "/"
						}
						return &u
					}(),
				}
				if d.IsDir() {
					out = append(out, fileData)
				} else {
					filesList = append(filesList, fileData)
				}
			}
			out = append(out, filesList...)
			return out
		}(),
	})
}

func (f *fileHandler) serveUploadTo(w http.ResponseWriter, r *http.Request, osPath string) error {
	mr, err := r.MultipartReader()
	if err != nil {
		return err
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else if part.FormName() == "file" {
			outPath := filepath.Join(osPath, filepath.Base(part.FileName()))
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer out.Close()
			if _, err := io.Copy(out, part); err != nil {
				return err
			}
		}
	}
	w.Header().Set("Location", r.URL.String())
	w.WriteHeader(303)
	return nil
}

func (f *fileHandler) createNewFolder(w http.ResponseWriter, r *http.Request, osPath string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	name := r.FormValue("name")
	if len(name) == 0 {
		w.WriteHeader(400)
		return fmt.Errorf("name must not be empty")
	}
	log.Println("try create folder:", osPath+osPathSeparator+name)
	err := os.Mkdir(osPath+osPathSeparator+name, 0665)
	if err != nil && !os.IsExist(err) {
		w.WriteHeader(400)
		log.Println("create folder", err)
		return err
	}
	log.Println("create folder:", name)
	w.Header().Set("Location", r.URL.String())
	w.WriteHeader(303)
	return nil
}

// ServeHTTP is http.Handler.ServeHTTP
func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s %s %s", f.path, r.RemoteAddr, r.Method, r.URL.String())
	urlPath := r.URL.Path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	urlPath = strings.TrimPrefix(urlPath, f.route)
	urlPath = strings.TrimPrefix(urlPath, "/"+f.route)

	osPath := strings.ReplaceAll(urlPath, "/", osPathSeparator)
	osPath = filepath.Clean(osPath)
	osPath = filepath.Join(f.path, osPath)
	info, err := os.Stat(osPath)
	switch {
	case os.IsNotExist(err):
		_ = f.serveStatus(w, r, http.StatusNotFound)
	case os.IsPermission(err):
		_ = f.serveStatus(w, r, http.StatusForbidden)
	case err != nil:
		fmt.Println(err)
		_ = f.serveStatus(w, r, http.StatusInternalServerError)
	case !f.allowDelete && r.Method == http.MethodDelete:
		_ = f.serveStatus(w, r, http.StatusForbidden)
	case !f.allowUpload && r.Method == http.MethodPost:
		_ = f.serveStatus(w, r, http.StatusForbidden)
	case !f.allowCreate && r.Method == http.MethodPut:
		_ = f.serveStatus(w, r, http.StatusForbidden)
	case r.URL.Query().Get(zipKey) != "":
		err := f.serveZip(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case r.URL.Query().Get(tarGzKey) != "":
		err := f.serveTarGz(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case f.allowUpload && info.IsDir() && r.Method == http.MethodPost:
		err := f.serveUploadTo(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case f.allowCreate && info.IsDir() && r.Method == http.MethodPut:
		err := f.createNewFolder(w, r, osPath)
		if err != nil {
			log.Println("error create folder:", err)
			w.Write([]byte(err.Error() + ".  "))
		}
	case f.allowDelete && !info.IsDir() && r.Method == http.MethodDelete:
		err := os.Remove(osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case info.IsDir():
		err := f.serveDir(w, r, osPath)
		if err != nil {
			fmt.Println(err)
			w.Write([]byte(err.Error() + "  "))
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	default:
		http.ServeFile(w, r, osPath)
	}
}
