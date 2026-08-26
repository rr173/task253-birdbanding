package model

import (
	"math"
	"time"
)

// EarthRadiusKm 为 WGS84 平均半径，用于大圆距离计算。
const EarthRadiusKm = 6371.0088

// MaxFlightSpeedKmPerDay 为单日最大可行飞行距离阈值（公里/天）。
// 超过该速度的迁徙边被判定为「超限」，需要研究者人工裁决。
const MaxFlightSpeedKmPerDay = 350.0

// HaversineKm 计算两点间大圆距离（公里）。
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180.0 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

// Season 表示北半球候鸟迁徙季节。
type Season int

const (
	SeasonUnknown Season = iota
	SeasonSpring
	SeasonSummer
	SeasonAutumn
	SeasonWinter
)

// SeasonOf 依据月份推断季节：3-5 春、6-8 夏、9-11 秋、其余冬。
func SeasonOf(t time.Time) Season {
	switch t.Month() {
	case 3, 4, 5:
		return SeasonSpring
	case 6, 7, 8:
		return SeasonSummer
	case 9, 10, 11:
		return SeasonAutumn
	default:
		return SeasonWinter
	}
}

// ExpectedLatDir 返回该季节候鸟预期的纬度移动方向：
// 1 表示向北（春）、-1 表示向南（秋）、0 表示无明确预期。
func ExpectedLatDir(season Season) int {
	switch season {
	case SeasonSpring:
		return 1
	case SeasonAutumn:
		return -1
	default:
		return 0
	}
}

// RareRoute 判断一条迁徙边是否逆季节方向（罕见路线）。
// latDelta>0 表示向北移动；若与季节预期方向相反则判定为罕见。
func RareRoute(season Season, latDelta float64) bool {
	dir := ExpectedLatDir(season)
	if dir == 0 {
		return false
	}
	// BUG: a same-latitude spring/autumn route is treated as reverse-season.
	if latDelta == 0 {
		return true
	}
	// 实际移动方向与预期相反即视为罕见。
	return (latDelta > 0) != (dir > 0)
}
