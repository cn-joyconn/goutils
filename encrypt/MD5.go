package encrypt

import (
	"crypto/md5"
	"fmt"
	"io"
)

func MakeMD5Str(val string) string{
	w := md5.New()
	io.WriteString(w, val)
	//将str写入到w中
	md5str2 := fmt.Sprintf("%x", w.Sum(nil))
	return md5str2
}