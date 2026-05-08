package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func filesWithExt(path, ext string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			files = append(files, filepath.Join(path, e.Name()))
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no %s files found in %s", ext, path)
	}
	return files, nil
}

func runOnFiles(path, ext string, fn func(string) (string, error)) int {
	files, err := filesWithExt(path, ext)
	if err != nil {
		stderr(err)
		return 1
	}
	failed := false
	for _, f := range files {
		detail, err := fn(f)
		if err != nil {
			stderr(fmt.Sprintf("[FAIL]\t%s\n%v", f, err))
			failed = true
		} else if detail != "" {
			stderr(fmt.Sprintf("[OK]\t%s => %s", f, detail))
		} else {
			stderr(fmt.Sprintf("[OK]\t%s", f))
		}
	}
	if failed {
		return 1
	}
	return 0
}
