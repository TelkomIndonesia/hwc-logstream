package main

import (
	"os"
	"strconv"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	ltsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2/region"
)

var (
	maxEndFromNow = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MAX_END_FROM_NOW"), time.Minute)
	maxFetchRange = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MAX_FETCH_RANGE"), 5*time.Minute)
	minFetchRange = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MIN_FETCH_RANGE"), time.Minute)
	maxLag        = strToDurationOrDefault(os.Getenv("HWC_LOGSTREAM_MAX_LAG"), 24*time.Hour)

	streamRoutine = int(strToIntOrDefault(os.Getenv("HWC_LOGSTREAM_ROUTINE"), 5))

	streamPosTag       = envOrDefault("HWC_LOGSTREAM_POSITITION_TAG", "x-hwc-logstream-pos")
	streamExclusionTag = envOrDefault("HWC_LOGSTREAM_EXCLUSION_TAG", "x-hwc-logstream-exclude")

	regionID = envOrDefault("HUAWEICLOUD_SDK_REGION_ID", "ap-southeast-4")
)

var additionalRegion = map[string]*region.Region{
	"ap-southeast-4": region.NewRegion("ap-southeast-4", "https://lts.ap-southeast-4.myhuaweicloud.com"),
}

func regionFromEnv() *region.Region {
	if r, ok := additionalRegion[regionID]; ok {
		return r
	}

	return ltsregion.ValueOf(regionID)
}

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
