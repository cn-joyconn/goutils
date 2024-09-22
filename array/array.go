package array

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

type chantype interface {
	int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		uintptr | float32 | float64 | string
}

// int 转 string
func IntArrToInString(i []int) string {
	s := make([]string, 0, len(i))
	for _, o := range i {
		s = append(s, strconv.Itoa(o))
	}
	return strings.Join(s, ",")
}

// int64 change string
func Int64ArrToInString(i64 []int64) string {
	s := make([]string, 0, len(i64))
	for _, o := range i64 {
		s = append(s, strconv.FormatInt(o, 10))
	}
	return strings.Join(s, ",")
}

// string arr change int
func StringArrToInString(s []string) string {
	return `"` + strings.Join(s, `","`) + `"`
}

func InArr[T chantype](s T, arr []T) bool {
	for _, val := range arr {
		if s == val {
			return true
		}
	}
	return false
}

// InStrArray 判断元素是否包含
func InStrArray(s string, arr []string) bool {
	// for _, val := range arr {
	// 	if s == val {
	// 		return true
	// 	}
	// }
	return InArr(s, arr)
}

// InIntArray 判断元素是否包含
func InIntArray(s int, arr []int) bool {

	return InArr(s, arr)
}
func RemoveDuplicate[T chantype](arr []T) (newArr []T) {
	newArr = make([]T, 0)
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
func RemoveDuplicateStr(arr []string) (newArr []string) {
	return RemoveDuplicate(arr)
}
func RemoveDuplicateInt(arr []int) (newArr []int) {
	return RemoveDuplicate(arr)
}
func RemoveDuplicateInt64(arr []int64) (newArr []int64) {
	return RemoveDuplicate(arr)
}
func Reverse[T chantype](l []T) (newArr []T) {
	for i := 0; i < int(len(l)/2); i++ {
		li := len(l) - i - 1
		l[i], l[li] = l[li], l[i]
	}
	return l
}
func ReverseStr(l []string) (newArr []string) {
	return Reverse(l)
}
func ReverseInt(l []int) (newArr []int) {
	return Reverse(l)
}
func ReverseInt64(l []int64) (newArr []int64) {
	return Reverse(l)
}

func Contain(obj interface{}, target interface{}) (bool, error) {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true, nil
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true, nil
		}
	}

	return false, errors.New("not in array")
}
