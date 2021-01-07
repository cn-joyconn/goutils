package strtool

import (
	"fmt"
	"strings"
	"testing"
)

func TestTrimRightSpace(t *testing.T) {
	str := "aaabc\n\t\r"
	rs := TrimRightSpace(str)
	fmt.Println(rs)
}
func TestMd5(t *testing.T) {
	str := RandomString(10)
	rs := Md5(str)
	fmt.Println(rs)
}
func TestRandomString(t *testing.T) {
	str := RandomString(5)
	fmt.Println(str)
}

func TestIsBlank(t *testing.T) {
	str := IsBlank(`\n\t\r `)
	ll := strings.Trim(`\n\t `, `\r\n\t `) 
	fmt.Println(str)
	fmt.Println(ll)
}