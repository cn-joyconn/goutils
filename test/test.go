package goutils

import (
	"fmt"
	"testing"

	"github.com/cn-joyconn/goutils/gosync"
)

type ABC interface {
	ID() int64
}
type AA struct {
	B int64
}

func (a *AA) ID() int64 {
	return a.B
}
func TestsyncMap(t *testing.T) {
	mmp := gosync.NewSyncMap[int, ABC]()
	mmp.Put(0, &AA{})
	v, o := mmp.Get(0)
	if o {
		fmt.Println(v.ID())
	}
	mmp.Remove(0)
	v, o = mmp.Get(0)
	if o {
		fmt.Println(v)
	}
}
