package array

import (
	"strconv"
	"strings"
)

//int 转 string
func IntArrToInString(i []int) string {
	s := make([]string, 0, len(i))
	for _, o := range i {
		s = append(s, strconv.Itoa(o))
	}
	return strings.Join(s, ",")
}

//int64 change string
func Int64ArrToInString(i64 []int64) string {
	s := make([]string, 0, len(i64))
	for _, o := range i64 {
		s = append(s, strconv.FormatInt(o, 10))
	}
	return strings.Join(s, ",")
}

//string arr change int
func StringArrToInString(s []string) string {
	return `"` + strings.Join(s, `","`) + `"`
}

//InStrArray 判断元素是否包含
func InStrArray(s string, arr []string) bool {
	for _, val := range arr {
		if s == val {
			return true
		}
	}
	return false
}

// InIntArray 判断元素是否包含
func InIntArray(s int, arr []int) bool {
	for _, val := range arr {
		if s == val {
			return true
		}
	}
	return false
}

func RemoveDuplicateStr(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}
func RemoveDuplicateInt(arr []int) (newArr []int) {
	newArr = make([]int, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}
func RemoveDuplicateInt64(arr []int64) (newArr []int64) {
	newArr = make([]int64, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

func ReverseStr(l []string)  {
    for i:=0; i < int(len(l)/2) ;i++{
        li := len(l) - i -1
        l[i],l[li] = l[li],l[i]
    }
}
func ReverseInt(l []int)  {
    for i:=0; i < int(len(l)/2) ;i++{
        li := len(l) - i -1
        l[i],l[li] = l[li],l[i]
    }
}
func ReverseInt64(l []int64)  {
    for i:=0; i < int(len(l)/2) ;i++{
        li := len(l) - i -1
        l[i],l[li] = l[li],l[i]
    }
}