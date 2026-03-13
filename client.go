package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// Client はDiscord API統合クライアント
type Client struct {
	token   string
	API     *API
	Gateway *Gateway
	logger  *slog.Logger

	// インタラクションハンドラー
	interactionHandler func(*Interaction)
}

// NewClient は新しいDiscord Clientを作成する
func NewClient(token string, logger *slog.Logger) *Client {
	api := NewAPI(token, logger)

	return &Client{
		token:   token,
		API:     api,
		Gateway: NewGateway(token, 0, logger), // intents=0 (インタラクションは常に受信される)
		logger:  logger,
	}
}

// OnInteraction はインタラクションハンドラーを登録する
func (c *Client) OnInteraction(handler func(*Interaction)) {
	c.interactionHandler = handler
}

// Open はGatewayに接続し、イベント購読を開始する
func (c *Client) Open() error {
	// Gateway URL を取得
	gatewayBot, err := c.API.GetGatewayBot()
	if err != nil {
		return fmt.Errorf("failed to get gateway URL: %w", err)
	}

	c.logger.Info("Got gateway URL", "url", gatewayBot.URL)

	// INTERACTION_CREATE イベントハンドラーを登録
	c.Gateway.On("INTERACTION_CREATE", func(data json.RawMessage) {
		var interaction Interaction
		if err := json.Unmarshal(data, &interaction); err != nil {
			c.logger.Error("Failed to unmarshal interaction", "error", err)
			return
		}

		if c.interactionHandler != nil {
			c.interactionHandler(&interaction)
		}
	})

	// Gateway に接続
	if err := c.Gateway.Connect(gatewayBot.URL); err != nil {
		return fmt.Errorf("failed to connect to gateway: %w", err)
	}

	return nil
}

// Close はGateway接続を閉じる
func (c *Client) Close() error {
	return c.Gateway.Close()
}

// BotUser はBot自身のユーザー情報を返す
func (c *Client) BotUser() *User {
	return c.Gateway.BotUser()
}

// RegisterCommands はスラッシュコマンドを一括登録する
func (c *Client) RegisterCommands(commands []*ApplicationCommand) error {
	user := c.BotUser()
	if user == nil {
		return fmt.Errorf("bot user not available (not connected?)")
	}

	registered, err := c.API.BulkOverwriteGlobalCommands(user.ID, commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	for _, cmd := range registered {
		c.logger.Info("Registered command", "name", cmd.Name, "id", cmd.ID)
	}

	return nil
}

// RespondToInteraction はインタラクションに応答する
func (c *Client) RespondToInteraction(interactionID, interactionToken string, response *InteractionResponse) error {
	return c.API.CreateInteractionResponse(interactionID, interactionToken, response)
}
