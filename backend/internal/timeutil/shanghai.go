package timeutil

import "time"

// shanghaiLocation keeps one stable UTC+8 location for all "server standard
// time" calculations. Asia/Shanghai currently has no daylight-saving changes,
// so a fixed zone is safer than depending on host tzdata availability.
var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// ShanghaiLocation returns the shared UTC+8 location used by LinuxDoSpace for
// all daily-reset and temporary-reward-expiry calculations.
func ShanghaiLocation() *time.Location {
	return shanghaiLocation
}

// ShanghaiDayKey returns the canonical YYYY-MM-DD label for the Shanghai day
// containing the provided timestamp.
func ShanghaiDayKey(value time.Time) string {
	return value.In(shanghaiLocation).Format("2006-01-02")
}

// ShanghaiDayBoundsUTC returns the inclusive start and exclusive end of the
// Shanghai-local day that contains the provided timestamp, both converted back
// into UTC so database comparisons can stay timezone-agnostic.
func ShanghaiDayBoundsUTC(value time.Time) (time.Time, time.Time) {
	localValue := value.In(shanghaiLocation)
	startLocal := time.Date(localValue.Year(), localValue.Month(), localValue.Day(), 0, 0, 0, 0, shanghaiLocation)
	endLocal := startLocal.Add(24 * time.Hour)
	return startLocal.UTC(), endLocal.UTC()
}

// NextShanghaiMidnightUTC returns the next Shanghai-local midnight converted
// into UTC. Temporary PoW rewards expire exactly at this boundary.
func NextShanghaiMidnightUTC(value time.Time) time.Time {
	_, end := ShanghaiDayBoundsUTC(value)
	return end
}
