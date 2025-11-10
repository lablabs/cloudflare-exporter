package main

import (
	"encoding/json"
	"strconv"
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

// NormalizePath removes query strings and fragments from a path to reduce cardinality
func NormalizePath(path string) string {
	// Strip query string
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	// Strip fragment
	if idx := strings.Index(path, "#"); idx != -1 {
		path = path[:idx]
	}
	// Handle empty path
	if path == "" {
		return "/"
	}
	return path
}

// ParseStatusFilter parses a status filter string and returns a function that checks if a status code should be included.
// Supports formats:
//   - "404,500" - specific status codes
//   - "300-499" - status code ranges
//   - "404,500-599" - combination of individual codes and ranges
//   - "" (empty) - all status codes
func ParseStatusFilter(filter string) (func(int) bool, error) {
	// Empty filter means accept all
	if strings.TrimSpace(filter) == "" {
		return func(status int) bool { return true }, nil
	}

	// Parse the filter into individual codes and ranges
	var codes []int
	var ranges [][2]int

	parts := strings.Split(filter, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a range (contains "-")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				log.Warnf("Invalid range format: %s, skipping", part)
				continue
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				log.Warnf("Invalid range start: %s, skipping", rangeParts[0])
				continue
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				log.Warnf("Invalid range end: %s, skipping", rangeParts[1])
				continue
			}

			ranges = append(ranges, [2]int{start, end})
		} else {
			// Individual code
			code, err := strconv.Atoi(part)
			if err != nil {
				log.Warnf("Invalid status code: %s, skipping", part)
				continue
			}
			codes = append(codes, code)
		}
	}

	// Return a function that checks if a status code matches the filter
	return func(status int) bool {
		// Check individual codes
		for _, code := range codes {
			if status == code {
				return true
			}
		}

		// Check ranges
		for _, r := range ranges {
			if status >= r[0] && status <= r[1] {
				return true
			}
		}

		return false
	}, nil
}
