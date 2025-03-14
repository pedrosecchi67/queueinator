package folder_handler

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Create directories that head to a file
func createDirectories(filePath string) error {
	// Extract the directory path from the file path
	dirPath := filepath.Dir(filePath)

	// Create all directories in the path with read/write/execute permissions (0755)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return nil
}

// get byte stream and obtain file names and sizes
func fileStructure(message []byte) ([]string, []int, []byte, error) {
	// get number of received files from message
	cut, message, _ := bytes.Cut(message, []byte(" "))
	var num_files int
	_, err := fmt.Sscanf(string(cut), "%d", &num_files)
	if err != nil {
		return nil, nil, nil, err
	}

	// get file names and sizes
	files := make([]string, num_files)
	file_sizes := make([]int, num_files)

	for i := range num_files {
		cut, message, _ = bytes.Cut(message, []byte("@"))
		files[i] = string(cut)

		cut, message, _ = bytes.Cut(message, []byte(" "))
		_, err := fmt.Sscanf(string(cut), "%d", &(file_sizes[i]))
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return files, file_sizes, message, nil
}

// Expand files from byte array into a directory
func Parse2Files(folder string, message []byte) error {
	files, file_sizes, message, err := fileStructure(message)
	if err != nil {
		return err
	}

	for i, file := range files {
		fsize := file_sizes[i]

		if fsize == -1 { // empty directory
			exists := false

			info, err := os.Stat(filepath.Join(folder, file))
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat directories: %w", err)
				}
			} else {
				exists = true
			}

			// It already exists. Let's just check that its a directory
			if exists {
				if !info.IsDir() {
					return fmt.Errorf("something that should be a directory is not a directory")
				}
			} else {
				err := os.MkdirAll(filepath.Join(folder, file), 0755)
				if err != nil {
					return fmt.Errorf("failed to create directories: %w", err)
				}
			}
		} else { // file
			cut := message[:fsize]
			message = message[fsize:]

			err := createDirectories(filepath.Join(folder, file))
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(folder, file), cut, 0666)
			if err != nil && err != io.EOF {
				return err
			}
		}
	}

	return nil
}
