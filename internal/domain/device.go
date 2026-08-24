package domain

// DeviceType identifies a class of measurement instrument.
type DeviceType string

const (
	DeviceProbe     DeviceType = "probe"     // 培养箱温湿度探头
	DeviceMolecular DeviceType = "molecular" // 分子检测仪
	DeviceMoisture  DeviceType = "moisture"  // 水分仪
	DevicePHMeter   DeviceType = "ph"        // pH 计
	DeviceCO2Meter  DeviceType = "co2"       // 二氧化碳计
)

// AttemptResult is the outcome of a device invocation recorded in the
// device_attempts ledger.
type AttemptResult string

const (
	AttemptAccepted     AttemptResult = "accepted"     // 设备接受并返回合格读数
	AttemptRejected     AttemptResult = "rejected"     // 设备拒绝
	AttemptDisconnected AttemptResult = "disconnected" // 设备断连
	AttemptTimeout      AttemptResult = "timeout"      // 设备超时
	AttemptFormatError  AttemptResult = "format_error" // 格式错误
)

// DeviceAttempt is a single auditable device invocation. Failed invocations
// only ever produce a DeviceAttempt and never a qualified reading or evidence.
type DeviceAttempt struct {
	DeviceType DeviceType
	DeviceID   DeviceID
	Object     string // 调用对象（指标或孔位）
	TaskID     TaskID
	Generation Generation
	At         LogicalTime
	ScriptSeq  int // 脚本序号
	Result     AttemptResult
	RetryCount int
	ErrorCode  ErrorCode
}
