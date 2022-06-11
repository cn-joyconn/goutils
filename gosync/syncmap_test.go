package gosync

import (
	"testing"
)

type testSt struct {
	A string
}

func TestSyncMap(t *testing.T) {
	mapObj := NewSyncMap[string, *testSt]()
	st := &testSt{
		A: "aaa",
	}
	mapObj.Put("a", st)
}
