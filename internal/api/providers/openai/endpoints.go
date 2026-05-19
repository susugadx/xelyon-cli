package openai

const (
	openAIAPIBaseURL                  = "https://api.openai.com"
	openAIChatCompletionsEndpointPath = "/v1/chat/completions"
	openAIResponsesEndpointPath       = "/v1/responses"
	defaultOpenAIURL                  = openAIAPIBaseURL + openAIChatCompletionsEndpointPath
	defaultOpenAIResponsesURL         = openAIAPIBaseURL + openAIResponsesEndpointPath
)
