package filetool

import (
	"os"
	"path"
)

// WriteBytesToFile saves content type '[]byte' to file by given path.
// It returns error when fail to finish operation.
func WriteBytesToFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(path.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}

// WriteBytesToFile saves content type '[]byte' to file by given path.
// It returns error when fail to finish operation.
func AppendBytesToFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(path.Dir(filePath), os.ModePerm)
	fw, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}

// WriteStringFile saves content type 'string' to file by given path.
// It returns error when fail to finish operation.
func WriteStringToFile(filePath string, s string) (int, error) {
	return WriteBytesToFile(filePath, []byte(s))
}
