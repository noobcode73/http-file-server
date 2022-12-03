package utils

import (
	zipper "archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func Zip(w io.Writer, path string, files []string) error {
	basePath := path
	addFile := func(w *zipper.Writer, path string, stat os.FileInfo) error {
		if stat.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		path, err = filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		zw, err := w.Create(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(zw, file); err != nil {
			return err
		}
		return w.Flush()
	}
	wZip := zipper.NewWriter(w)
	defer func() {
		if err := wZip.Close(); err != nil {
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
					return addFile(wZip, path, info)
				})
				if err != nil {
					fmt.Println(err)
					return err
				}
			} else {
				err = addFile(wZip, fPath, info)
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
		return addFile(wZip, path, info)
	})
}
