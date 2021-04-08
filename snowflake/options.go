package snowflake



// SnowOptions .在https://github.com/yitter/idgenerator-go基础上进行订制修改
// 时间戳存储范围2^(64-(WorkerIdBitLength+SeqBitLength))-1, 例(64-(WorkerIdBitLength+SeqBitLength))=31时,大约68年就会用完
// 这个一旦定义且开始生成id后千万不要改了 不然可能会生成相同的id
type SnowOptions struct {
	BaseTime          uint64  // 基础时间（ms单位），不能超过当前系统时间(默认2021-1-1)
	WorkerId          uint16 // 机器码，必须由外部设定，最大值 2^WorkerIdBitLength-1
	WorkerIdBitLength byte   // 机器码位长，默认值6，取值范围 [1, 20]（要求：序列数位长+机器码位长不超过23）
	SeqBitLength      byte   // 序列数位长，默认值6，取值范围 [3, 22]（要求：序列数位长+机器码位长不超过23）
	MaxSeqNumber      uint32 // 最大序列数（含），设置范围 [MinSeqNumber, 2^SeqBitLength-1]，默认值0，表示最大序列数取最大值（2^SeqBitLength-1]）
	MinSeqNumber      uint32 // 最小序列数（含），默认值5，取值范围 [5, MaxSeqNumber]，每毫秒的前5个序列数对应编号0-4是保留位，其中1-4是时间回拨相应预留位，0是手工新值预留位
	TopOverCostCount  uint32 // 最大漂移次数（含），默认2000，推荐范围500-10000（与计算能力有关）
}

// NewSnowOptions .
func NewSnowOptions(workerId uint16) *SnowOptions {
	return &SnowOptions{
		WorkerId:          workerId,
		BaseTime:          1612108800000,//(默认2021-1-1)
		WorkerIdBitLength: 6,
		SeqBitLength:      6,
		MaxSeqNumber:      0,
		MinSeqNumber:      5,
		TopOverCostCount:  2000,
	}
}
