package safety

import (
	"regexp"
	"strings"
)

var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(https?://)[^:@/\s]+:[^@/\s]+@`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b`),
	regexp.MustCompile(`(?i)\b(glpat-[A-Za-z0-9\-_]{20,})\b`),
	regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret|secret_key|secret)\s*[:=]\s*[A-Za-z0-9/+=]{40}\b`),
	regexp.MustCompile(`(?i)(token|key|password|secret|passwd|api_key|apikey)\s*[:=]\s*\S+`),
}

// SanitizeText masks common credential patterns before printing or logging.
func SanitizeText(s string) string {
	out := s
	for _, re := range redactPatterns {
		out = re.ReplaceAllStringFunc(out, func(m string) string {
			if strings.HasPrefix(strings.ToLower(m), "http") {
				return re.ReplaceAllString(m, "${1}[REDACTED]@")
			}
			if idx := strings.IndexAny(m, ":="); idx >= 0 {
				return m[:idx+1] + "[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return out
}
