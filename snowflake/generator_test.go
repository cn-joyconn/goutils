package snowflake

import (
	"fmt"
	"testing"
	"time"
)

func TestLogin(t *testing.T) {
	var options = NewSnowOptions(1)
	options.WorkerIdBitLength = 11
	options.SeqBitLength = 11
	//options.WorkerIdBitLength = 6
	//options.SeqBitLength = 6
	//options.BaseTime = time.Date(2020, 2, 20, 2, 20, 2, 20, time.UTC).UnixNano() / 1e6
	InitGenerator(options)

	var times = 50000
	for {
		var begin = time.Now().UnixNano() / 1e3
		for i := 0; i < times; i++ {
			fmt.Println(NextId())
			// idgen.NextId()
		}

		var end = time.Now().UnixNano() / 1e3
		fmt.Println(end - begin)
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
}
