package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func GetTimeRange() (now time.Time, now1mAgo time.Time) {
	now = time.Now().Add(-time.Duration(viper.GetInt("scrape_delay")) * time.Second).UTC()
	s := 60 * time.Second
	now = now.Truncate(s)
	now1mAgo = now.Add(-60 * time.Second)

	return now, now1mAgo
}

func jsonStringToMap(fields string) (map[string]interface{}, error) {
	var extraFields map[string]interface{}
	err := json.Unmarshal([]byte(fields), &extraFields)
	return extraFields, err
}

var (
	numericIDPattern = regexp.MustCompile(`^[0-9]+$`)
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	hexIDPattern     = regexp.MustCompile(`^[0-9a-fA-F]{8,}$`)
)

func normalizePath(path string) string {
	if path == "" || path == "/" {
		return path
	}

	path = strings.Split(path, "?")[0]

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		if numericIDPattern.MatchString(segment) {
			segments[i] = ":id"
		} else if uuidPattern.MatchString(segment) {
			segments[i] = ":uuid"
		} else if hexIDPattern.MatchString(segment) {
			segments[i] = ":id"
		}
	}

	return strings.Join(segments, "/")
}
