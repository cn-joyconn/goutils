package snowflake

import (
	"sync"
	"time"
)

// SnowWorker .
type SnowWorker struct {
	BaseTime          uint64  //基础时间
	WorkerId          uint16 //机器码
	WorkerIdBitLength byte   //机器码位长
	SeqBitLength      byte   //自增序列数位长
	MaxSeqNumber      uint32 //最大序列数（含）
	MinSeqNumber      uint32 //最小序列数（含）
	TopOverCostCount  uint32 //最大漂移次数
	_TimestampShift   byte
	_CurrentSeqNumber uint32

	_LastTimeTick           uint64
	_TurnBackTimeTick       uint64
	_TurnBackIndex          byte
	_IsOverCost             bool
	_OverCostCountInOneTerm uint32
	_GenCountInOneTerm      uint32
	_TermIndex              uint32

	sync.Mutex
}

// NewSnowWorker .
func NewSnowWorker(options *SnowOptions) *SnowWorker {
	var workerIdBitLength byte
	var seqBitLength byte
	var maxSeqNumber uint32

	// 1.BaseTime
	var baseTime uint64
	if options.BaseTime != 0 {
		baseTime = options.BaseTime
	} else {
		baseTime = 1582136402000
	}

	// 2.WorkerIdBitLength
	if options.WorkerIdBitLength == 0 {
		workerIdBitLength = 6
	} else {
		workerIdBitLength = options.WorkerIdBitLength
	}

	// 3.WorkerId
	var workerId = options.WorkerId

	// 4.SeqBitLength
	if options.SeqBitLength == 0 {
		seqBitLength = 6
	} else {
		seqBitLength = options.SeqBitLength
	}

	// 5.MaxSeqNumber
	if options.MaxSeqNumber <= 0 {
		maxSeqNumber = (1 << seqBitLength) - 1
	} else {
		maxSeqNumber = options.MaxSeqNumber
	}

	// 6.MinSeqNumber
	var minSeqNumber = options.MinSeqNumber

	// 7.Others
	var topOverCostCount = options.TopOverCostCount
	if topOverCostCount == 0 {
		topOverCostCount = 2000
	}

	timestampShift := (byte)(workerIdBitLength + seqBitLength)
	currentSeqNumber := minSeqNumber

	return &SnowWorker{
		BaseTime:          baseTime,
		WorkerIdBitLength: workerIdBitLength,
		WorkerId:          workerId,
		SeqBitLength:      seqBitLength,
		MaxSeqNumber:      maxSeqNumber,
		MinSeqNumber:      minSeqNumber,
		TopOverCostCount:  topOverCostCount,
		_TimestampShift:   timestampShift,
		_CurrentSeqNumber: currentSeqNumber,
		_LastTimeTick:           0,
		_TurnBackTimeTick:       0,
		_TurnBackIndex:          0,
		_IsOverCost:             false,
		_OverCostCountInOneTerm: 0,
		_GenCountInOneTerm:      0,
		_TermIndex:              0,
	}
}

// DoGenIDAction .
func (m1 *SnowWorker) DoGenIdAction(arg *OverCostActionArg) {

}

func (m1 *SnowWorker) BeginOverCostAction(useTimeTick uint64) {

}

func (m1 *SnowWorker) EndOverCostAction(useTimeTick uint64) {
	if m1._TermIndex > 10000 {
		m1._TermIndex = 0
	}
}

func (m1 *SnowWorker) BeginTurnBackAction(useTimeTick uint64) {

}

func (m1 *SnowWorker) EndTurnBackAction(useTimeTick uint64) {

}

func (m1 *SnowWorker) NextOverCostId() uint64 {
	currentTimeTick := m1.GetCurrentTimeTick()
	if currentTimeTick > m1._LastTimeTick {
		m1.EndOverCostAction(currentTimeTick)
		m1._LastTimeTick = currentTimeTick
		m1._CurrentSeqNumber = m1.MinSeqNumber
		m1._IsOverCost = false
		m1._OverCostCountInOneTerm = 0
		m1._GenCountInOneTerm = 0
		return m1.CalcId(m1._LastTimeTick)
	}
	if m1._OverCostCountInOneTerm >= m1.TopOverCostCount {
		m1.EndOverCostAction(currentTimeTick)
		m1._LastTimeTick = m1.GetNextTimeTick()
		m1._CurrentSeqNumber = m1.MinSeqNumber
		m1._IsOverCost = false
		m1._OverCostCountInOneTerm = 0
		m1._GenCountInOneTerm = 0
		return m1.CalcId(m1._LastTimeTick)
	}
	if m1._CurrentSeqNumber > m1.MaxSeqNumber {
		m1._LastTimeTick++
		m1._CurrentSeqNumber = m1.MinSeqNumber
		m1._IsOverCost = true
		m1._OverCostCountInOneTerm++
		m1._GenCountInOneTerm++

		return m1.CalcId(m1._LastTimeTick)
	}

	m1._GenCountInOneTerm++
	return m1.CalcId(m1._LastTimeTick)
}

// NextNormalID .
func (m1 *SnowWorker) NextNormalId() uint64 {
	currentTimeTick := m1.GetCurrentTimeTick()
	if currentTimeTick < m1._LastTimeTick {
		if m1._TurnBackTimeTick < 1 {
			m1._TurnBackTimeTick = m1._LastTimeTick - 1
			m1._TurnBackIndex++
			// 每毫秒序列数的前5位是预留位，0用于手工新值，1-4是时间回拨次序
			// 最多4次回拨（防止回拨重叠）
			if m1._TurnBackIndex > 4 {
				m1._TurnBackIndex = 1
			}
			m1.BeginTurnBackAction(m1._TurnBackTimeTick)
		}

		// time.Sleep(time.Duration(1) * time.Millisecond)
		return m1.CalcTurnBackId(m1._TurnBackTimeTick)
	}

	// 时间追平时，_TurnBackTimeTick清零
	if m1._TurnBackTimeTick > 0 {
		m1.EndTurnBackAction(m1._TurnBackTimeTick)
		m1._TurnBackTimeTick = 0
	}

	if currentTimeTick > m1._LastTimeTick {
		m1._LastTimeTick = currentTimeTick
		m1._CurrentSeqNumber = m1.MinSeqNumber
		return m1.CalcId(m1._LastTimeTick)
	}

	if m1._CurrentSeqNumber > m1.MaxSeqNumber {
		m1.BeginOverCostAction(currentTimeTick)
		m1._TermIndex++
		m1._LastTimeTick++
		m1._CurrentSeqNumber = m1.MinSeqNumber
		m1._IsOverCost = true
		m1._OverCostCountInOneTerm = 1
		m1._GenCountInOneTerm = 1

		return m1.CalcId(m1._LastTimeTick)
	}

	return m1.CalcId(m1._LastTimeTick)
}

// CalcID .
func (m1 *SnowWorker) CalcId(useTimeTick uint64) uint64 {
	result := uint64(useTimeTick<<m1._TimestampShift) + uint64(m1.WorkerId<<m1.SeqBitLength) + uint64(m1._CurrentSeqNumber)
	m1._CurrentSeqNumber++
	return result
}

// CalcTurnBackID .
func (m1 *SnowWorker) CalcTurnBackId(useTimeTick uint64) uint64 {
	result := uint64(useTimeTick<<m1._TimestampShift) + uint64(m1.WorkerId<<m1.SeqBitLength) + uint64(m1._TurnBackIndex)
	m1._TurnBackTimeTick--
	return result
}

// GetCurrentTimeTick .
func (m1 *SnowWorker) GetCurrentTimeTick() uint64 {
	var millis = uint64(time.Now().UnixNano() / 1e6)
	return millis - m1.BaseTime
}

// GetNextTimeTick .
func (m1 *SnowWorker) GetNextTimeTick() uint64 {
	tempTimeTicker := m1.GetCurrentTimeTick()
	for tempTimeTicker <= m1._LastTimeTick {
		tempTimeTicker = m1.GetCurrentTimeTick()
	}
	return tempTimeTicker
}

// NextId .
func (m1 *SnowWorker) NextId() uint64 {
	m1.Lock()
	defer m1.Unlock()
	if m1._IsOverCost {
		return m1.NextOverCostId()
	} else {
		return m1.NextNormalId()
	}
}
