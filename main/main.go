package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := "C:/Users/cbw/Documents/notebrew"
	root = filepath.FromSlash(root) + string(os.PathSeparator)
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		fmt.Println(strings.TrimPrefix(path, root))
		return nil
	})
	fmt.Println("root:", root)
}
