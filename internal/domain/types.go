// Package domain holds the stable shared value types and identifiers used
// across every bounded component of the MycoCycle grow-bag transfer gate.
package domain

// Identifiers and scalar value types shared by all components.

type TaskID string
type Generation int
type OperationID string

type BagBatch string        // 菌包批号
type BagPosition string     // 抽检袋位
type RackID string          // 培养架位
type ProbeID string         // 探头编号
type PersonID string        // 复核/抽检人员
type QualificationID string // 资质编号
type DeviceID string

type LogicalTime int64 // 逻辑时钟，单调递增
type DayAge int        // 培养日龄

// Metric identifies a physico-chemical quantity recorded with fixed-point
// integer arithmetic.
type Metric string

const (
	MetricMoisture    Metric = "moisture"    // 含水率
	MetricPH          Metric = "ph"          // pH
	MetricTemperature Metric = "temperature" // 温度
	MetricHumidity    Metric = "humidity"    // 湿度
	MetricCO2         Metric = "co2"         // 二氧化碳
)
