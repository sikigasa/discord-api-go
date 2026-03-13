package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const discordAPIBase = "https://discord.com/api/v10"

// API はDiscord APIクライアント
type API struct {
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAPI は新しいDiscord APIクライアントを作成する
func NewAPI(token string, logger *slog.Logger) *API {
	return &API{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetGatewayBot は Gateway Bot URL を取得する
func (r *API) GetGatewayBot() (*GatewayBotResponse, error) {
	body, err := r.doRequest("GET", "/gateway/bot", nil)
	if err != nil {
		return nil, err
	}

	var resp GatewayBotResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway bot response: %w", err)
	}

	return &resp, nil
}

// BulkOverwriteGlobalCommands はグローバルコマンドを一括登録する
func (r *API) BulkOverwriteGlobalCommands(appID string, commands []*ApplicationCommand) ([]*ApplicationCommand, error) {
	path := fmt.Sprintf("/applications/%s/commands", appID)
	body, err := r.doRequest("PUT", path, commands)
	if err != nil {
		return nil, err
	}

	var result []*ApplicationCommand
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commands response: %w", err)
	}

	return result, nil
}

// CreateInteractionResponse はインタラクションに応答する
func (r *API) CreateInteractionResponse(interactionID, interactionToken string, response *InteractionResponse) error {
	path := fmt.Sprintf("/interactions/%s/%s/callback", interactionID, interactionToken)
	_, err := r.doRequest("POST", path, response)
	return err
}

// ─── Internal ───────────────────────────────────────────

func (r *API) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, discordAPIBase+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+r.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "DiscordBot (todoapp-discordbot, 1.0)")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		r.logger.Error("Discord API error",
			"method", method,
			"path", path,
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("Discord API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
