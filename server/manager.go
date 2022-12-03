package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (f *FileHandler) serveStatus(w http.ResponseWriter, r *http.Request, status int) error {
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

// ServeHTTP is http.Handler.ServeHTTP
func (f *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		log.Println(err)
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
	case f.allowUpload && info.IsDir() && r.Method == http.MethodPost && r.URL.Query().Has("new") == false:
		err := f.serveUploadTo(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case f.allowCreate && info.IsDir() && r.Method == http.MethodPost && r.URL.Query().Has("new"):
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
			log.Println(err)
			w.Write([]byte(err.Error() + "  "))
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	default:
		http.ServeFile(w, r, osPath)
	}
}
