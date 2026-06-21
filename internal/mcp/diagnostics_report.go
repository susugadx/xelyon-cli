package mcp

// SummaryStatus はレポート全体の代表 status を返す。
func (r DiagnosticReport) SummaryStatus() DiagnosticStatus {
	if r.HasFailures() {
		return DiagnosticStatusFail
	}
	for _, check := range r.allChecks() {
		if check.Status == DiagnosticStatusWarn {
			return DiagnosticStatusWarn
		}
	}
	return DiagnosticStatusOK
}

// HasFailures はレポートに fail check があるかを返す。
func (r DiagnosticReport) HasFailures() bool {
	for _, check := range r.allChecks() {
		if check.Status == DiagnosticStatusFail {
			return true
		}
	}
	return false
}

func (r *DiagnosticReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func (r *DiagnosticServerReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     detail,
		Suggestion: suggestion,
	})
}

func (r DiagnosticReport) allChecks() []DiagnosticCheck {
	checks := append([]DiagnosticCheck{}, r.Checks...)
	for _, server := range r.Servers {
		checks = append(checks, server.Checks...)
	}
	return checks
}
