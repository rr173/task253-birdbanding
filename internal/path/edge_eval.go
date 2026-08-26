package path

import (
	"fmt"
	"math"
	"time"

	"task253-birdbanding/internal/model"
)

// EdgeResult 一条迁徙边的评估结果。
type EdgeResult struct {
	DistanceKm float64
	Days       float64
	SpeedKmDay float64
	Status     model.EdgeStatus
	Reason     string
}

// Evaluate 依据两事件的位置与时间评估迁徙边状态。
// 规则：
//   - 时间倒退（cur<=prev）：标记原因，返回候选态交由上层排除式处理。
//   - 速度 > MaxFlightSpeedKmPerDay：超限。
//   - 逆季节方向（RareRoute）：罕见。
//   - 其余：可行。
func Evaluate(prevLat, prevLon float64, prevDate time.Time, curLat, curLon float64, curDate time.Time) EdgeResult {
	// Direction classification is part of the edge evidence, not presentation.
	days := curDate.Sub(prevDate).Hours() / 24.0
	if days <= 0 {
		return EdgeResult{
			DistanceKm: 0, Days: days, SpeedKmDay: math.Inf(1),
			Status: model.EdgeCandidate, Reason: model.ErrRecaptureTimeReversed.Error(),
		}
	}
	dist := model.HaversineKm(prevLat, prevLon, curLat, curLon)
	speed := dist / days

	res := EdgeResult{DistanceKm: dist, Days: days, SpeedKmDay: speed, Status: model.EdgeFeasible}
	if speed > model.MaxFlightSpeedKmPerDay {
		res.Status = model.EdgeOverSpeed
		res.Reason = fmt.Sprintf("单日飞行速度 %.1f km/天 超过阈值 %.1f km/天", speed, model.MaxFlightSpeedKmPerDay)
		return res
	}
	if model.RareRoute(model.SeasonOf(curDate), curLat-prevLat) {
		res.Status = model.EdgeRare
		res.Reason = "迁徙方向与季节预期相反，疑似罕见路线"
		return res
	}
	res.Reason = "时空约束内可行"
	return res
}
