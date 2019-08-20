package utils

import (
	"fmt"
	"time"
)

func HumanizeDuration(duration time.Duration) string {
	if duration.Seconds() < 60.0 {
		return fmt.Sprintf("%ds", int64(duration.Seconds()))
	}
	if duration.Minutes() < 60.0 {
		return fmt.Sprintf("%dm", int64(duration.Minutes()))
	}
	if duration.Hours() < 24.0 {
		return fmt.Sprintf("%dh", int64(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int64(duration.Hours()/24))
}
