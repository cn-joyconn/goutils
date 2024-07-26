package filetool

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// ReadFileToBytes reads data type '[]byte' from file by given path.
// It returns error when fail to finish operation.
func ReadFileToBytes(filePath string) ([]byte, error) {
	// b, err := ioutil.ReadFile(filePath)
	// if err != nil {
	// 	return []byte(""), err
	// }
	// return b, nil

	file, err := os.Open(filePath)
	if err != nil {
		return []byte(""), err
	}
	defer file.Close() // 确保文件在函数结束时关闭

	reader := bufio.NewReader(file)
	content, err := io.ReadAll(reader)
	if err != nil {
		return []byte(""), err
	}
	return content, err
}
func ReadFileToLineBytes(filePath string) ([][]byte, error) {
	result := make([][]byte, 0)
	if IsExist(filePath) && IsFile(filePath) {
		fi, err := os.Open(filePath)
		if err == nil {
			defer fi.Close()
			br := bufio.NewReader(fi)
			for {
				a, _, c := br.ReadLine()
				if c == io.EOF {
					break
				}
				result = append(result, a)
			}
		}
	}

	return result, nil
}

// ReadFileToString reads data type 'string' from file by given path.
// It returns error when fail to finish operation.
func ReadFileToString(filePath string) (string, error) {
	b, err := ReadFileToBytes(filePath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// 读取文件里的数据,去掉空格,换行,制表符,回车
func ReadFileToStringNoLn(filePath string) (string, error) {
	str, err := ReadFileToString(filePath)
	if err != nil {
		return "", err
	}
	str = strings.Trim(string(str), " ")
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\r", "", -1)
	str = strings.Replace(str, "\t", "", -1)
	return str, nil
}
