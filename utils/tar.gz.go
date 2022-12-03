package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func TarGz(w io.Writer, path string, files []string) error {
	basePath := path
	addFile := func(w *tar.Writer, path string, stat os.FileInfo) error {
		if stat.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		header := new(tar.Header)
		path, err = filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		header.Name = path
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		if err := w.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(w, file); err != nil {
			return err
		}
		return w.Flush()
	}
	wGzip := gzip.NewWriter(w)
	wTar := tar.NewWriter(wGzip)
	defer func() {
		if err := wTar.Close(); err != nil {
			log.Println(err)
		}
		if err := wGzip.Close(); err != nil {
			log.Println(err)
		}
	}()

	if len(files) > 0 {
		for _, item := range files {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			fPath := path + string(filepath.Separator) + item
			info, err := os.Lstat(fPath)
			if err != nil {
				fmt.Println(err)
				return err
			}
			if info.IsDir() {
				err = filepath.Walk(fPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					return addFile(wTar, path, info)
				})
				if err != nil {
					fmt.Println(err)
					return err
				}
			} else {
				err = addFile(wTar, fPath, info)
				if err != nil {
					fmt.Println(err)
					return err
				}
			}
		}
		return nil
	}

	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return addFile(wTar, path, info)
	})
}
