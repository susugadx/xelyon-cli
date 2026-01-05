package api

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
)

const deepseekURL = "https://api.deepseek.com/chat/completions"

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool      `json:"stream"`
}

type Delta struct {
    Content string `json:"content"`
}

type StreamChoice struct {
    Delta Delta `json:"delta"`
}

type StreamResponse struct {
    Choices []StreamChoice `json:"choices"`
}

type Choice struct {
    Message Message `json:"message"`
}

type ChatResponse struct {
    Choices []Choice `json:"choices"`
}

func AskDeepSeekStream(query string, context string) (string, error) {
    apiKey := os.Getenv("DEEPSEEK_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("DEEPSEEK_API_KEY not set")
    }

    systemPrompt := "You are an excellent programming assistant. Answer based on the provided context. When modifying code, always present the complete code wrapped in triple backticks."

    userContent := query
    if context != "" {
        userContent = fmt.Sprintf("## Context:\n%s\n\n## Question:\n%s", context, query)
    }

    reqBody := ChatRequest{
        Model: "deepseek-chat",
        Messages: []Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userContent},
        },
        Stream: true,
    }

    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", deepseekURL, bytes.NewBuffer(jsonBody))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("API error: %s", string(body))
    }

    var fullResponse strings.Builder
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" {
                break
            }

            var streamResp StreamResponse
            if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
                continue
            }

            if len(streamResp.Choices) > 0 {
                content := streamResp.Choices[0].Delta.Content
                fmt.Print(content)
                fullResponse.WriteString(content)
            }
        }
    }

    fmt.Println()
    return fullResponse.String(), nil
}
