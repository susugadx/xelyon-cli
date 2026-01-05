package api

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

const baseURL = "https://xelyon-backend-265806801386.asia-northeast1.run.app"

type RAGRequest struct {
    Query  string `json:"query"`
    UserID string `json:"user_id"`
    Model  string `json:"model"`
    TopK   int    `json:"top_k"`
}

type RAGResult struct {
    ID            string  `json:"id"`
    Content       string  `json:"content"`
    DocumentID    string  `json:"document_id"`
    DocumentTitle string  `json:"document_title"`
    DocumentType  string  `json:"document_type"`
    Similarity    float64 `json:"similarity"`
}

type RAGResponse struct {
    Results []RAGResult `json:"results"`
    Query   string      `json:"query"`
    Count   int         `json:"count"`
}

func SearchRAG(query string, userID string, topK int) (*RAGResponse, error) {
    reqBody := RAGRequest{
        Query:  query,
        UserID: userID,
        Model:  "small",
        TopK:   topK,
    }

    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }

    resp, err := http.Post(baseURL+"/api/rag/search", "application/json", bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("API error: %s", string(body))
    }

    var ragResp RAGResponse
    if err := json.Unmarshal(body, &ragResp); err != nil {
        return nil, err
    }

    return &ragResp, nil
}
