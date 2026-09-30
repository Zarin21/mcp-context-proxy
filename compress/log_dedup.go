package compress

import (
	"fmt"
	"regexp"
	"strings"
)

// LogDeduplicator collapses repeated log lines
type LogDeduplicator struct{}

// normalizeRegex matches sequences of digits (including in timestamps, IPs, durations)
var normalizeRegex = regexp.MustCompile(`\d+`)

// timestampPrefixRegex matches common log timestamp prefixes
var timestampPrefixRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[^\s]*\s*`)

func (d *LogDeduplicator) normalize(line string) string {
	// Strip the timestamp prefix entirely — it varies per line
	stripped := timestampPrefixRegex.ReplaceAllString(line, "")
	// Replace all remaining numbers
	return normalizeRegex.ReplaceAllString(stripped, "#")
}

// Dedup takes a string of log lines and returns a deduplicated version.
// Consecutive lines that normalize to the same pattern are collapsed into
// a single representative line with a "(repeated N times)" annotation.
func (d *LogDeduplicator) Dedup(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return text
	}

	var result []string
	count := 1
	lastLine := lines[0]
	lastNormalized := d.normalize(lines[0])

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			// Preserve empty lines
			if count > 1 {
				result = append(result, fmt.Sprintf("%s (repeated %d times)", lastLine, count))
			} else {
				result = append(result, lastLine)
			}
			result = append(result, "")
			count = 1
			if i+1 < len(lines) {
				lastLine = lines[i+1]
				lastNormalized = d.normalize(lines[i+1])
				i++
			}
			continue
		}

		normalized := d.normalize(line)

		if normalized == lastNormalized {
			count++
		} else {
			if count > 1 {
				result = append(result, fmt.Sprintf("%s (repeated %d times)", lastLine, count))
			} else {
				result = append(result, lastLine)
			}
			lastLine = line
			lastNormalized = normalized
			count = 1
		}
	}

	// Flush the last group
	if count > 1 {
		result = append(result, fmt.Sprintf("%s (repeated %d times)", lastLine, count))
	} else {
		result = append(result, lastLine)
	}

	return strings.Join(result, "\n")
}
