package model

import "testing"

func TestStatusMachinesProtectTerminalStates(t *testing.T) {
	if !BatchCanTransition(BatchDraft, BatchReview) || BatchCanTransition(BatchSealed, BatchReview) {
		t.Fatalf("批次状态机边界错误")
	}
	if !EventCanTransition(EventPending, EventValid) || EventCanTransition(EventValid, EventExcluded) {
		t.Fatalf("事件终态不应继续流转")
	}
	if !EdgeCanTransition(EdgeRare, EdgeConfirmed) || EdgeCanTransition(EdgeConfirmed, EdgeCandidate) {
		t.Fatalf("迁徙边确认态边界错误")
	}
	if !VersionCanTransition(VersionShared, VersionFrozen) || !VersionCanTransition(VersionFrozen, VersionSuperseded) {
		t.Fatalf("路径版本冻结流转错误")
	}
	if !VersionFrozen.IsImmutable() || VersionDraft.IsImmutable() {
		t.Fatalf("版本不可变属性错误")
	}
}
