package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type kratosError struct {
	ID    string `json:"id"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	} `json:"error"`
}

func HandleKratosError(w http.ResponseWriter, r *http.Request, kratosClient *KratosFlowClient) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing error id", http.StatusBadRequest)
		return
	}

	msg, code := fetchKratosError(r.Context(), kratosClient, r, id)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_ = RenderPage(w, PageData{
		Title:       "エラー",
		Description: msg,
		FlowType:    "error",
		Flow:        KratosFlow{Messages: []KratosMessage{{Text: msg, Type: "error"}}},
	})
}

func fetchKratosError(ctx context.Context, c *KratosFlowClient, r *http.Request, id string) (string, int) {
	endpoint := c.apiBaseURL + "/self-service/errors?id=" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "エラー情報の取得に失敗しました", http.StatusInternalServerError
	}
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Kratosへの接続に失敗しました: %s", err), http.StatusBadGateway
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var ke kratosError
	if err := json.Unmarshal(body, &ke); err != nil {
		return strings.TrimSpace(string(body)), http.StatusInternalServerError
	}

	msg := ke.Error.Message
	if ke.Error.Reason != "" {
		msg += ": " + ke.Error.Reason
	}
	if msg == "" {
		msg = "不明なエラーが発生しました"
	}
	return msg, http.StatusOK
}
