package provider

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

// validateSchedule enforces the cron_expression XOR interval_seconds invariant.
func validateSchedule(cronExpr types.String, intervalSecs types.Int64) error {
	cronSet := !cronExpr.IsNull() && !cronExpr.IsUnknown()
	intervalSet := !intervalSecs.IsNull() && !intervalSecs.IsUnknown()
	switch {
	case cronSet && intervalSet:
		return errors.New("exactly one of cron_expression or interval_seconds must be set, not both")
	case !cronSet && !intervalSet:
		return errors.New("exactly one of cron_expression or interval_seconds must be set")
	}
	return nil
}

// appendAPIError translates client.APIError into Terraform diagnostics,
// mapping known 422 error codes to helpful attribute-level messages.
func appendAPIError(diags *diag.Diagnostics, action string, err error) {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		diags.AddError("API error while "+action, err.Error())
		return
	}

	switch apiErr.StatusCode {
	case http.StatusUnauthorized:
		diags.AddError(
			"Authentication failed",
			"The SteadyCron API returned 401. Check that STEADYCRON_API_KEY is set and uses a valid Full-scope key.",
		)
	case http.StatusTooManyRequests:
		diags.AddError(
			"Rate limit exceeded",
			"The SteadyCron API returned 429 after all retries. Reduce provider parallelism or wait before retrying.",
		)
	case http.StatusUnprocessableEntity:
		msg := fmt.Sprintf("Error %s: %s", action, apiErr.Message)
		switch apiErr.Code {
		case "plan_job_limit_exceeded":
			msg = "Plan job limit exceeded: upgrade your SteadyCron plan or delete unused jobs."
		case "frequency_below_plan_floor":
			msg = "Schedule frequency is below the minimum allowed by your plan. Use a longer interval or cron expression."
		case "plan_account_too_new":
			msg = "Account is too new to create HTTP jobs with external URLs. Wait 24 hours."
		}
		diags.AddError("Validation error while "+action, msg)
	default:
		diags.AddError(
			fmt.Sprintf("Error while %s (HTTP %d)", action, apiErr.StatusCode),
			apiErr.Error(),
		)
	}
}

func boolPtr(b bool) *bool { return &b }

func int64Ptr(i int64) *int64 { return &i }

// stringPtrOrEmpty returns the pointed-to string, or "" if nil.
// Use this for Optional+Computed fields with Default:"" to avoid null/empty drift
// when the API returns null for an empty string field.
func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// subMicrosecondRe matches ISO 8601 timestamps that have more than 6 fractional-second digits.
var subMicrosecondRe = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\.(\d{7,})([+-]\d{2}:\d{2}|Z)`)

// normalizeTimestamp truncates sub-microsecond precision from API timestamps so that
// POST responses (7 digits) and GET responses (6 digits) produce identical strings in state.
func normalizeTimestamp(s string) string {
	return subMicrosecondRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := subMicrosecondRe.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		frac := strings.TrimRight(sub[2][:6], "0")
		if frac == "" {
			return sub[1] + sub[3]
		}
		return sub[1] + "." + frac + sub[3]
	})
}
