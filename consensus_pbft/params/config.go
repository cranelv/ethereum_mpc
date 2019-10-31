package params

import "time"

type General struct {
	N uint32		//max consenter count
	F uint32		//max. number of faults we can tolerate
	ViewChangePeriod   uint64        // period between automatic view changes
	BatchSize	int
}
type LogInfo struct {
	K uint64		//checkpoint period
	LogMultiplier uint64            // use this value to calculate log size : k*logMultiplier
}
type DebugInfo struct {
	Byzantine bool
}
type TimeInfo struct {
	Request				time.Duration
	Resendviewchange	time.Duration
	Viewchange			time.Duration
	Nullrequest			time.Duration
	Broadcast			time.Duration
	BatchTimeout		time.Duration
}
type Config struct {
	General
	LogInfo
	DebugInfo
	TimeInfo
}

func DefaultConfig()*Config {
	return &Config{
		General{
			N:4,
			F:1,
			ViewChangePeriod : 0,
			BatchSize : 500,
		},
		LogInfo{
			K:10,
			LogMultiplier:4,
		},
		DebugInfo{
			Byzantine:false,
		},
		TimeInfo{
			Request:2*time.Second,
			Resendviewchange:2*time.Second,
			Viewchange:2*time.Second,
			Nullrequest:0*time.Second,
			Broadcast:time.Second,
			BatchTimeout:time.Second,
		},
	}
}
