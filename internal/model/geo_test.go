package model

import (
	"math"
	"testing"
	"time"
)

func TestGeoAndSeasonRules(t *testing.T) {
	if distance := HaversineKm(0, 0, 0, 1); math.Abs(distance-111.195) > 0.2 {
		t.Fatalf("一度经度距离异常: %.3f", distance)
	}
	if got := SeasonOf(time.Date(2025, time.October, 1, 0, 0, 0, 0, time.UTC)); got != SeasonAutumn {
		t.Fatalf("十月应为秋季: %v", got)
	}
	if !RareRoute(SeasonAutumn, 5) || RareRoute(SeasonAutumn, -5) {
		t.Fatalf("秋季逆纬度方向判定错误")
	}
}

func TestNewLocationRequiresPrecision(t *testing.T) {
	if _, err := NewLocation("loc-1", "缺精度", 1, 2, 0); err != ErrLocationPrecisionMissing {
		t.Fatalf("缺少地点精度应被拒绝: %v", err)
	}
	loc, err := NewLocation("loc-2", "有效地点", 1, 2, 25)
	if err != nil || loc.PrecisionM != 25 {
		t.Fatalf("有效地点创建失败: loc=%+v err=%v", loc, err)
	}
}
