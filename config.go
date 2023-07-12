package main

import (
	"os"
	"strconv"
	"time"
)

var maxEndFromNow = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MAX_END_FROM_NOW"), time.Minute)
var maxFetchRange = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MAX_FETCH_RANGE"), 5*time.Minute)
var minFetchRange = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MIN_FETCH_RANGE"), time.Minute)
var streamRoutine = int(strToIntOrDefault(os.Getenv("HWC_LOGSTREAM_ROUTINE"), 5))
var streamPosTag = envOrDefault("HWC_LOGSTREAM_POSITITION_TAG", "x-hwc-logstream-pos")
var streamExclusionTag = envOrDefault("HWC_LOGSTREAM_EXCLUSION_TAG", "x-hwc-logstream-exclude")

func envOrDefault(name, def string) string {
	v, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	return v
}

func strToDurationOrDefault(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func strToIntOrDefault(s string, i int64) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return i
	}
	return v
}
