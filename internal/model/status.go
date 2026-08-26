package model

// 四类状态机：观测批次、环志/重捕事件、迁徙边、路径版本。

// BatchStatus 观测批次状态。
type BatchStatus string

const (
	BatchDraft     BatchStatus = "录入中"
	BatchReview    BatchStatus = "待复核"
	BatchPublished BatchStatus = "已发布"
	BatchSealed    BatchStatus = "封存"
)

// BatchCanTransition 校验批次状态流转合法性。
func BatchCanTransition(from, to BatchStatus) bool {
	switch from {
	case BatchDraft:
		return to == BatchReview || to == BatchSealed
	case BatchReview:
		return to == BatchPublished
	case BatchPublished:
		return to == BatchSealed
	case BatchSealed:
		return false
	default:
		return false
	}
}

// EventStatus 环志/重捕事件状态。
type EventStatus string

const (
	EventPending   EventStatus = "待校验"
	EventValid     EventStatus = "有效"
	EventConflict  EventStatus = "身份冲突"
	EventExcluded  EventStatus = "排除"
)

// EventCanTransition 校验事件状态流转合法性。
func EventCanTransition(from, to EventStatus) bool {
	if from == EventPending {
		return to == EventValid || to == EventConflict || to == EventExcluded
	}
	// 已终态不可再流转（校正环号会先重置为待校验）。
	return false
}

// EdgeStatus 迁徙边状态。
type EdgeStatus string

const (
	EdgeCandidate  EdgeStatus = "候选"
	EdgeFeasible   EdgeStatus = "可行"
	EdgeOverSpeed  EdgeStatus = "超限"
	EdgeRare       EdgeStatus = "罕见"
	EdgeConfirmed  EdgeStatus = "确认"
)

// EdgeCanTransition 校验迁徙边状态流转合法性。
func EdgeCanTransition(from, to EdgeStatus) bool {
	switch from {
	case EdgeCandidate:
		return to == EdgeFeasible || to == EdgeOverSpeed || to == EdgeRare
	case EdgeFeasible, EdgeOverSpeed, EdgeRare:
		// 可确认为路径组成，也可回退为候选以便研究者重新裁决。
		return to == EdgeConfirmed || to == EdgeCandidate
	case EdgeConfirmed:
		return false
	default:
		return false
	}
}

// VersionStatus 路径版本状态。
type VersionStatus string

const (
	VersionDraft      VersionStatus = "草稿"
	VersionShared     VersionStatus = "共享"
	VersionFrozen     VersionStatus = "冻结"
	VersionSuperseded VersionStatus = "替代"
)

// VersionCanTransition 校验路径版本状态流转合法性。冻结后仅可被替代。
func VersionCanTransition(from, to VersionStatus) bool {
	switch from {
	case VersionDraft:
		return to == VersionShared
	case VersionShared:
		return to == VersionFrozen
	case VersionFrozen:
		return to == VersionSuperseded
	case VersionSuperseded:
		return false
	default:
		return false
	}
}

// IsImmutable 判断给定状态是否不可变（用于冻结版本拒绝修改）。
func (s VersionStatus) IsImmutable() bool {
	return s == VersionFrozen || s == VersionSuperseded
}
