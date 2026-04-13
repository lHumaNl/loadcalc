package engine

import "math"

// RampUpConfig holds the calculated ramp-up scheduler parameters for LRE PC.
type RampUpConfig struct {
	BatchSize   int // Vusers per iteration (>=1)
	IntervalSec int // seconds between batches (>=1)
	Iterations  int // total iterations including last partial
	LastBatch   int // Vusers in last iteration (may be < BatchSize)
	ActualSec   int // actual ramp-up duration = Iterations * IntervalSec
}

// CalculateRampUp finds optimal batch size and interval for LRE PC scheduler.
// delta = number of Vusers to add, rampupSec = target ramp-up duration.
func CalculateRampUp(delta, rampupSec int) RampUpConfig {
	if delta <= 0 {
		return RampUpConfig{}
	}
	if rampupSec <= 0 {
		return RampUpConfig{
			BatchSize:   delta,
			IntervalSec: 0,
			Iterations:  1,
			LastBatch:   delta,
			ActualSec:   0,
		}
	}

	bestDev := math.MaxInt64
	var best RampUpConfig

	for batch := 1; batch <= delta; batch++ {
		iterations := (delta + batch - 1) / batch // ceil(delta/batch)
		interval := int(math.Round(float64(rampupSec) / float64(iterations)))
		if interval < 1 {
			interval = 1
		}
		actual := iterations * interval
		dev := actual - rampupSec
		if dev < 0 {
			dev = -dev
		}
		if dev < bestDev {
			bestDev = dev
			lastBatch := delta % batch
			if lastBatch == 0 {
				lastBatch = batch
			}
			best = RampUpConfig{
				BatchSize:   batch,
				IntervalSec: interval,
				Iterations:  iterations,
				LastBatch:   lastBatch,
				ActualSec:   actual,
			}
		}
		// On tie: keep smaller batch (first found wins), so no update on ==
	}

	return best
}
