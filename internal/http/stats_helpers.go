package httpapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"pixia-panel/internal/store"
)

func fillLast24(list []store.StatisticsFlow, userID int64) []store.StatisticsFlow {
	if len(list) >= 24 {
		return list
	}
	result := make([]store.StatisticsFlow, len(list))
	copy(result, list)

	startHour := time.Now().Hour()
	if len(result) > 0 {
		startHour = parseHour(result[len(result)-1].Time) - 1
	}

	for len(result) < 24 {
		if startHour < 0 {
			startHour = 23
		}
		result = append(result, store.StatisticsFlow{
			UserID:    userID,
			Flow:      0,
			TotalFlow: 0,
			Time:      fmt.Sprintf("%02d:00", startHour),
		})
		startHour--
	}
	return result
}

func parseHour(val string) int {
	if val == "" {
		return time.Now().Hour()
	}
	if strings.Contains(val, ":") {
		parts := strings.Split(val, ":")
		if len(parts) > 0 {
			if h, err := strconv.Atoi(parts[0]); err == nil {
				return h
			}
		}
	}
	return time.Now().Hour()
}
