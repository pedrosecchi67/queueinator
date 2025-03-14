package folder_handler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Obtain list with all files in a directory and its subirectories
func listFilesRecursively(root string) ([]string, error) {
	var files []string

	// Walk the directory tree
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Handle errors (e.g., permission issues)
		}

		// Make the path relative to the root directory
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the directory: %w", err)
	}

	return files, nil
}

// generate message with header and folder contents
func GenFolderMessage(folder string, buffer_size int) ([]byte, error) {
	files, err := listFilesRecursively(folder)
	if err != nil {
		return nil, err
	}

	// message with number of files
	header := strconv.Itoa(len(files))
	contents := make([]byte, 0)

	for _, file := range files {
		info, err := os.Stat(filepath.Join(folder, file))
		if err != nil {
			return nil, err
		}

		if info.IsDir() { // folder
			header += " " + file + "@-1"
		} else { // file
			fobj, err := os.Open(filepath.Join(folder, file))
			if err != nil {
				return nil, err
			}
			defer fobj.Close()

			buff := make([]byte, buffer_size)
			n, err := fobj.Read(buff)
			if err != nil && err != io.EOF {
				return nil, err
			}

			header += " " + file + "@" + strconv.Itoa(n)
			contents = append(contents, buff[:n]...)
		}
	}

	header += " "
	// join header bytes
	message := append([]byte(header), contents...)

	if len(message) > buffer_size {
		return nil, fmt.Errorf("passed message surpasses buffer size %d Mb", buffer_size/1024/1024)
	}

	return message, nil
}
