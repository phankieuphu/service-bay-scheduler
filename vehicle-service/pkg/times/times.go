package times

import (
	"fmt"
	"strconv"
	"time"
)

const (
	RFC3339Nano = time.RFC3339Nano // 2025-12-10T09:51:53.88139027Z
)

func ConvertFormat(input string) string {
	t, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	return t.Format(RFC3339Nano)
}

func OffsetDays(date time.Time, day int) time.Time {
	return date.AddDate(0, 0, day)
}

var utc7 = time.FixedZone("UTC+7", 7*60*60) // 7 hours in seconds

func FormatTimeRFC3339(date string) string {
	parsedOutput, err := time.Parse("2006-01-02T15:04:05", date)
	if err != nil {
		return ""
	}
	return parsedOutput.Format("2006-01-02T15:04:05.000Z")
}

func WeekRange(date string) (start time.Time, end time.Time) {
	t, err := time.Parse(time.RFC3339Nano, date)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	year, month, day := t.In(utc7).Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, utc7).UTC()
	daysToMonday := int(startOfDay.Weekday())
	if daysToMonday == -1 {
		daysToMonday = 6
	}
	start = startOfDay.AddDate(0, 0, -daysToMonday).UTC()
	sunday := start.AddDate(0, 0, 6)
	end = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 999999999, utc7).UTC()
	return start, end
}

func MonthRange(date string) (start time.Time, end time.Time) {
	t, err := time.Parse(time.RFC3339Nano, date)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	year, month, _ := t.In(utc7).Date()
	start = time.Date(year, month, 1, 0, 0, 0, 0, utc7).UTC()
	end = time.Date(year, month+1, 0, 0, 0, 0, 0, utc7).UTC().Add(-1 * time.Nanosecond)
	return start, end
}

func DayRange(date string) (start time.Time, end time.Time) {
	t, err := time.Parse(time.RFC3339Nano, date)
	if err != nil {
		return time.Time{}, time.Time{}
	}
	year, month, day := t.In(utc7).Date()
	start = time.Date(year, month, day, 0, 0, 0, 0, utc7).UTC()
	startOfNextDay := start.AddDate(0, 0, 1)
	end = startOfNextDay.Add(-time.Nanosecond)
	return start, end
}

func FormatUnixMilli(t interface{}) int64 {
	date, _ := ConvertToUnixMilli(t)
	return date
}

func ConvertToUnixMilli(input interface{}) (int64, error) {
	switch v := input.(type) {
	case time.Time:
		return v.UnixMilli(), nil
	case string:
		return parseStringToUnixMilli(v)
	case int64:
		if v < 1e12 {
			return v * 1000, nil
		}
		return v, nil
	case int:
		if int64(v) < 1e12 {
			return int64(v) * 1000, nil
		}
		return int64(v), nil
	case float64:
		return int64(v * 1000), nil
	default:
		return 0, fmt.Errorf("unsupported input type: %T", input)
	}
}

func parseStringToUnixMilli(dateStr string) (int64, error) {
	formats := []string{
		time.RFC3339Nano,                // "2006-01-02T15:04:05.999999999Z07:00"
		time.RFC3339,                    // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05.000Z",      // ISO8601 with milliseconds
		"2006-01-02T15:04:05+00:00",     // ISO8601 with UTC+0 (colon format)
		"2006-01-02T15:04:05+0000",      // ISO8601 with UTC+0 (compact format)
		"2006-01-02T15:04:05.000+00:00", // ISO8601 with milliseconds and UTC+0 (colon format)
		"2006-01-02T15:04:05.000+0000",  // ISO8601 with milliseconds and UTC+0 (compact format)
		"2006-01-02T15:04:05",           // ISO8601 without timezone
		"2006-01-02 15:04:05",           // Common datetime format
		"2006-01-02 15:04:05.000",       // Datetime with milliseconds
		"2006-01-02 15:04:05.000000",    // Datetime with microseconds
		"2006-01-02 15:04:05 +00:00",    // Datetime with UTC+0 (colon format)
		"2006-01-02 15:04:05 +0000",     // Datetime with UTC+0 (compact format)
		time.RFC1123,                    // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC1123Z,                   // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC822,                     // "02 Jan 06 15:04 MST"
		time.RFC822Z,                    // "02 Jan 06 15:04 -0700"
		time.RFC850,                     // "Monday, 02-Jan-06 15:04:05 MST"
		"2006-01-02",                    // Date only
		"02/01/2006",                    // DD/MM/YYYY
		"01/02/2006",                    // MM/DD/YYYY
		"2006/01/02",                    // YYYY/MM/DD
		"2006-01-02 15:04:05 MST",       // Datetime with timezone name
		"2006-01-02 15:04:05 -0700",     // Datetime with timezone offset
		"2006-01-02T15:04:05-0700",      // ISO8601 with timezone offset
		"2006-01-02T15:04:05.000-0700",  // ISO8601 with milliseconds and timezone
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t.UnixMilli(), nil
		}
		lastErr = err
	}

	if timestamp, err := strconv.ParseInt(dateStr, 10, 64); err == nil {
		if timestamp < 1e12 {
			return timestamp * 1000, nil
		}
		return timestamp, nil
	}

	if timestamp, err := strconv.ParseFloat(dateStr, 64); err == nil {
		return int64(timestamp * 1000), nil
	}

	return 0, fmt.Errorf("unable to parse time string: %s, last error: %v", dateStr, lastErr)
}

func ParseUnixMilli(unixMilli int64) time.Time {
	seconds := unixMilli / 1000
	nanos := (unixMilli % 1000) * 1_000_000
	return time.Unix(seconds, nanos).UTC()
}

func GetTimeNowUTCUnixMilli() int64 {
	return time.Now().UnixMilli()
}
