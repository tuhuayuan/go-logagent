package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// LogEvent struct of log event.
type LogEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Tags      []string               `json:"tags,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

// Formate-Type
var (
	reTime = regexp.MustCompile(`\${\+([^}]+)}`)
	reVar  = regexp.MustCompile(`\${([\w@]+)}`)
)

const (
	timeFormat = `2006-01-02T15:04:05.999`
	dateFormat = `2006-01-02`
)

// AddTag add tags to event.
func (le *LogEvent) AddTag(tags ...string) {
	for _, tag := range tags {
		ftag := le.Format(tag)
		found := false
		for _, ele := range le.Tags {
			if ele == ftag {
				found = true
				break
			}
		}

		if !found {
			le.Tags = append(le.Tags, ftag)
		}
	}
}

// Marshal return json bytes (packed).
func (le LogEvent) Marshal(readable bool) (data []byte, err error) {
	event := le.GetMap()
	if readable {
		return json.MarshalIndent(event, "", "  ")
	}
	return json.Marshal(event)
}

// Get interface{} from string key
func (le LogEvent) Get(field string) (v interface{}) {
	switch field {
	case "@timestamp":
		v = le.Timestamp
	case "@date":
		v = le.Timestamp
	case "message":
		v = le.Message
	default:
		v = le.Extra[field]
	}
	return
}

// GetString get string value from key
func (le LogEvent) GetString(field string) (v string) {
	switch field {
	case "@timestamp":
		v = le.Timestamp.UTC().Format(timeFormat)
	case "@date":
		v = le.Timestamp.UTC().Format(dateFormat)
	case "message":
		v = le.Message
	default:
		if value, ok := le.Extra[field]; ok {
			v = fmt.Sprintf("%v", value)
		}
	}
	return
}

// GetMap map[string]interface of LogEvent instance.
func (le LogEvent) GetMap() map[string]interface{} {
	event := map[string]interface{}{
		"@timestamp": le.Timestamp.UTC().Format(timeFormat),
	}
	if le.Message != "" {
		event["message"] = le.Message
	}
	if len(le.Tags) > 0 {
		event["tags"] = le.Tags
	}
	for key, value := range le.Extra {
		event[key] = value
	}
	return event
}

// FormatWithEnv fill environment var
func FormatWithEnv(text string) (result string) {
	result = text

	matches := reVar.FindAllStringSubmatch(result, -1)
	for _, submatches := range matches {
		field := submatches[1]
		value := os.Getenv(field)
		if value != "" {
			result = strings.Replace(result, submatches[0], value, -1)
		}
	}

	return
}

// FormatWithTime format string with current time, ex: ${+2006-01-02} see time.Time documents.
func FormatWithTime(text string) (result string) {
	result = text

	matches := reTime.FindAllStringSubmatch(result, -1)
	for _, submatches := range matches {
		value := time.Now().Format(submatches[1])
		result = strings.Replace(result, submatches[0], value, -1)
	}

	return
}

// Format format input string with field values of logevent.
func (le LogEvent) Format(format string) (out string) {
	out = format

	out = FormatWithTime(out)

	matches := reVar.FindAllStringSubmatch(out, -1)
	for _, submatches := range matches {
		field := submatches[1]
		value := le.GetString(field)
		if value != "" {
			out = strings.Replace(out, submatches[0], value, -1)
		}
	}

	out = FormatWithEnv(out)

	return
}
