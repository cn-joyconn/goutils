package snowflake

import (
	"strconv"
	"sync"
	"time"
)


type SnowGenerator struct {
	Options              *SnowOptions
	SnowWorker           SnowWorker
	IdGeneratorException SnowException
}
// NewIdGenerator 初始化一个生成器
func NewIdGenerator(options *SnowOptions) *SnowGenerator {
	if options == nil {
		panic("dig.Options error.")
	}

	// 1.BaseTime
	minTime := int64(631123200000) // time.Now().AddDate(-30, 0, 0).UnixNano() / 1e6
	if options.BaseTime < minTime || options.BaseTime > time.Now().UnixNano()/1e6 {
		panic("BaseTime error.")
	}

	// 2.WorkerIdBitLength
	if options.WorkerIdBitLength <= 0 {
		panic("WorkerIdBitLength error.(range:[1, 20])")
	}
	if options.WorkerIdBitLength+options.SeqBitLength > 23 {
		panic("error：WorkerIdBitLength + SeqBitLength <= 23")
	}

	// 3.WorkerId
	maxWorkerIdNumber := uint16(1<<options.WorkerIdBitLength) - 1
	if maxWorkerIdNumber == 0 {
		maxWorkerIdNumber = 63
	}
	if options.WorkerId < 0 || options.WorkerId > maxWorkerIdNumber {
		panic("WorkerId error. (range:[0, " + strconv.FormatUint(uint64(maxWorkerIdNumber), 10) + "]")
	}

	// 4.SeqBitLength
	if options.SeqBitLength < 3 || options.SeqBitLength > 22 {
		panic("SeqBitLength error. (range:[3, 22])")
	}

	// 5.MaxSeqNumber
	maxSeqNumber := uint32(1<<options.SeqBitLength) - 1
	if maxSeqNumber == 0 {
		maxSeqNumber = 63
	}
	if options.MaxSeqNumber < 0 || options.MaxSeqNumber > maxSeqNumber {
		panic("MaxSeqNumber error. (range:[1, " + strconv.FormatUint(uint64(maxSeqNumber), 10) + "]")
	}

	// 6.MinSeqNumber
	if options.MinSeqNumber < 5 || options.MinSeqNumber > maxSeqNumber {
		panic("MinSeqNumber error. (range:[5, " + strconv.FormatUint(uint64(maxSeqNumber), 10) + "]")
	}

	snowWorker :=  NewSnowWorker(options)
	time.Sleep(time.Duration(500) * time.Microsecond)

	return &SnowGenerator{
		Options:    options,
		SnowWorker: *snowWorker,
	}
}
// NextId 生成一个新的ID
func (sg *SnowGenerator)NextId()int64{
	return sg.SnowWorker.NextId()
}


var singletonMutex sync.Mutex
var singletonGenerator *SnowGenerator

// InitGenerator 初始化全局生成器
func InitGenerator(options *SnowOptions) {
	singletonMutex.Lock()
	singletonGenerator = NewIdGenerator(options)
	singletonMutex.Unlock()
}

// NextId 利用全局生成器生成一个新的ID
func NextId() int64 {
	if singletonGenerator == nil {
		singletonMutex.Lock()
		defer singletonMutex.Unlock()
		if singletonGenerator == nil {
			options := NewSnowOptions(1)
			singletonGenerator = NewIdGenerator(options)
		}
	}

	return singletonGenerator.NextId()
}

