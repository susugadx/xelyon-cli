package bedrock

import (
	"strings"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

func (p *Provider) clearLastRequestID() {
	if p == nil {
		return
	}
	p.lastRequestID = ""
}

func (p *Provider) captureRequestIDFromInvokeOutput(output *bedrockruntime.InvokeModelWithResponseStreamOutput) {
	if p == nil || output == nil {
		return
	}
	requestID, ok := awsmiddleware.GetRequestIDMetadata(output.ResultMetadata)
	if !ok {
		return
	}
	p.lastRequestID = strings.TrimSpace(requestID)
}

func (p *Provider) captureRequestIDFromConverseOutput(output *bedrockruntime.ConverseStreamOutput) {
	if p == nil || output == nil {
		return
	}
	requestID, ok := awsmiddleware.GetRequestIDMetadata(output.ResultMetadata)
	if !ok {
		return
	}
	p.lastRequestID = strings.TrimSpace(requestID)
}

func (p *Provider) lastBedrockRequestID() string {
	if p == nil {
		return ""
	}
	return p.lastRequestID
}
