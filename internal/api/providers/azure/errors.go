package azure

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type azureHTTPErrorContext struct {
	Deployment  string
	ToolPayload bool
}

func handleAzureResponsesHTTPError(resp *http.Response, spinner *uiruntime.Spinner, context azureHTTPErrorContext) error {
	if spinner != nil {
		spinner.Stop()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("azure OpenAI API error (status %d): unable to read response body - %v", resp.StatusCode, err)
	}

	detail := azureSanitizedErrorDetail(body, resp.StatusCode)
	advice := azureHTTPErrorAdvice(resp, detail, context)
	if detail == "" {
		return fmt.Errorf("azure OpenAI API error (status %d): %s", resp.StatusCode, advice)
	}
	return fmt.Errorf("azure OpenAI API error (status %d): %s Detail: %s", resp.StatusCode, advice, detail)
}

func azureSanitizedErrorDetail(body []byte, statusCode int) string {
	if len(body) == 0 {
		return "empty response body"
	}
	message := api.SanitizeErrorMessage(body, statusCode).Error()
	return strings.TrimPrefix(message, fmt.Sprintf("API error (%d): ", statusCode))
}

func azureHTTPErrorAdvice(resp *http.Response, detail string, context azureHTTPErrorContext) string {
	if context.ToolPayload && azureErrorMentionsToolPayload(detail) {
		return "tool payload was rejected. If this Azure deployment does not support function calling, set AZURE_OPENAI_FUNCTION_CALLING=0 and rerun the request."
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("authentication failed. Check %s, %s, or %s and make sure the credential belongs to this Azure OpenAI resource.", apiKeyEnv, authTokenEnv, authTokenCommandEnv)
	case http.StatusForbidden:
		return "authorization failed. Check Azure OpenAI permissions, Entra ID role assignment, subscription access, and deployment access for this resource."
	case http.StatusNotFound:
		return azureNotFoundAdvice(context.Deployment)
	case http.StatusTooManyRequests:
		return azureRateLimitAdvice(resp)
	default:
		return "request failed. Check the Azure OpenAI error detail and rerun `xelyon doctor azure --smoke` after correcting the configuration."
	}
}

func azureNotFoundAdvice(deployment string) string {
	deployment = strings.TrimSpace(deployment)
	if deployment == "" {
		return fmt.Sprintf("resource was not found. Check %s uses the resource v1 URL ending in %s and that the Azure deployment exists.", baseURLEnv, azureOpenAIBasePath)
	}
	return fmt.Sprintf("resource was not found. Check %s uses the resource v1 URL ending in %s and that Azure deployment %q exists in that resource.", baseURLEnv, azureOpenAIBasePath, deployment)
}

func azureRateLimitAdvice(resp *http.Response) string {
	message := "rate limit, quota, or Azure capacity was exceeded after retries. Check Azure OpenAI quota/capacity for the deployment or retry later."
	if retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After")); retryAfter != "" {
		message += " Retry-After: " + retryAfter + "."
	}
	return message
}

func azureErrorMentionsToolPayload(detail string) bool {
	detail = strings.ToLower(detail)
	return strings.Contains(detail, "tool") ||
		strings.Contains(detail, "function") ||
		strings.Contains(detail, "tool_choice")
}
