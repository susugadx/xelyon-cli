package rawoutputs

import "regexp"

const sensitiveStreamTailBytes = 16 * 1024

var (
	privateKeyBeginPattern = regexp.MustCompile(`(?is)-----BEGIN [^-]*PRIVATE KEY-----`)
	urlUserInfoPattern     = regexp.MustCompile(`(?i)https?://[^\s/@]+@`)
)

type sensitiveStreamDetector struct {
	tail string
}

func (d *sensitiveStreamDetector) Write(chunk []byte) bool {
	if len(chunk) == 0 {
		return false
	}
	window := d.tail + string(chunk)
	if LooksSensitiveContent(window) ||
		privateKeyBeginPattern.MatchString(window) ||
		urlUserInfoPattern.MatchString(window) {
		return true
	}
	d.tail = sensitiveTail(window)
	return false
}

func sensitiveTail(value string) string {
	if len(value) <= sensitiveStreamTailBytes {
		return value
	}
	return value[len(value)-sensitiveStreamTailBytes:]
}
