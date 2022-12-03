package server

import (
	"github.com/dastoori/higgs"
	"github.com/muller2002/http-file-server/utils"
	"io"
	"path/filepath"
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
	newFolderKey     = "new"
	tarGzKey         = "tar.gz"
	tarGzValue       = "true"
	tarGzContentType = "application/x-tar+gzip"
	zipKey           = "zip"
	zipValue         = "true"
	zipContentType   = "application/zip"
	osPathSeparator  = string(filepath.Separator)
)

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

type FileHandler struct {
	route          string
	path           string
	allowUpload    bool
	allowDelete    bool
	allowCreate    bool
	customTemplate string
	noAllowHidden  bool
}

func (f *FileHandler) serveTarGz(w http.ResponseWriter, r *http.Request, path string) error {
	w.Header().Set("Content-Type", tarGzContentType)
	name := filepath.Base(path) + ".tar.gz"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return utils.TarGz(w, path)
}

func (f *FileHandler) serveZip(w http.ResponseWriter, r *http.Request, osPath string) error {
	w.Header().Set("Content-Type", zipContentType)
	name := filepath.Base(osPath) + ".zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return utils.Zip(w, osPath)
}

func (f *FileHandler) serveDir(w http.ResponseWriter, r *http.Request, osPath string) error {
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

func (f *FileHandler) serveUploadTo(w http.ResponseWriter, r *http.Request, osPath string) error {
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

func (f *FileHandler) createNewFolder(w http.ResponseWriter, r *http.Request, osPath string) error {
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
