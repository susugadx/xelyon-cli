package gemini

import "strings"

type sseFinalizeOutput interface {
	Append(text string)
	Len() int
	Response() string
}

type sseFinalizeBuilderOutput struct {
	builder *strings.Builder
}

func newSSEFinalizeBuilderOutput(builder *strings.Builder) sseFinalizeOutput {
	return &sseFinalizeBuilderOutput{builder: builder}
}

func (o *sseFinalizeBuilderOutput) Append(text string) {
	o.builder.WriteString(text)
}

func (o *sseFinalizeBuilderOutput) Len() int {
	return o.builder.Len()
}

func (o *sseFinalizeBuilderOutput) Response() string {
	return o.builder.String()
}
