package doctor

import "time"

func timeNowUTC() string       { return time.Now().UTC().Format(time.RFC3339) }
func timeIn7DaysUTC() string   { return time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339) }
