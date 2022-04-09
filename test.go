package goutils

import (
	"fmt"
	"testing"

	"github.com/cn-joyconn/goutils/syncmap"
)

func TestsyncMap(t *testing.T) {
	mmp := syncmap.NewSyncMap[int, int]()
	mmp.Put(0, 1)
	v, o := mmp.Get(0)
	if o {
		fmt.Println(v)
	}
	mmp.Remove(0)
	v, o = mmp.Get(0)
	if o {
		fmt.Println(v)
	}
}
