package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Gateway はDiscord Gateway WebSocket接続を管理する
type Gateway struct {
	token   string
	intents int
	logger  *slog.Logger

	conn      *websocket.Conn
	seq       atomic.Int64
	sessionID string
	resumeURL string
	botUser   *User

	heartbeatInterval time.Duration
	lastHeartbeatACK  atomic.Bool

	handlers map[string][]func(json.RawMessage)
	mu       sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewGateway は新しいGatewayを作成する
func NewGateway(token string, intents int, logger *slog.Logger) *Gateway {
	return &Gateway{
		token:    token,
		intents:  intents,
		logger:   logger,
		handlers: make(map[string][]func(json.RawMessage)),
		done:     make(chan struct{}),
	}
}

// On はイベントハンドラーを登録する
func (g *Gateway) On(event string, handler func(json.RawMessage)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.handlers[event] = append(g.handlers[event], handler)
}

// BotUser はBot自身のユーザー情報を返す
func (g *Gateway) BotUser() *User {
	return g.botUser
}

// Connect はGatewayに接続する
func (g *Gateway) Connect(gatewayURL string) error {
	g.ctx, g.cancel = context.WithCancel(context.Background())

	url := gatewayURL + "?v=10&encoding=json"
	g.logger.Info("Connecting to gateway", "url", url)

	conn, _, err := websocket.Dial(g.ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway: %w", err)
	}
	g.conn = conn

	// Read size limit を大きめに設定
	g.conn.SetReadLimit(1 << 20) // 1MB

	// Hello を受信
	if err := g.readHello(); err != nil {
		g.conn.Close(websocket.StatusGoingAway, "hello failed")
		return fmt.Errorf("failed to read hello: %w", err)
	}

	// Identify を送信
	if err := g.sendIdentify(); err != nil {
		g.conn.Close(websocket.StatusGoingAway, "identify failed")
		return fmt.Errorf("failed to identify: %w", err)
	}

	// Ready を待機
	if err := g.waitForReady(); err != nil {
		g.conn.Close(websocket.StatusGoingAway, "ready failed")
		return fmt.Errorf("failed to wait for ready: %w", err)
	}

	// Heartbeat ループを開始
	go g.heartbeatLoop()

	// イベントリスナーを開始
	go g.listenLoop()

	return nil
}

// Close はGateway接続を閉じる
func (g *Gateway) Close() error {
	g.cancel()
	if g.conn != nil {
		return g.conn.Close(websocket.StatusNormalClosure, "bot shutting down")
	}
	return nil
}

// Done は接続が閉じられたときに閉じるチャネルを返す
func (g *Gateway) Done() <-chan struct{} {
	return g.done
}

// ─── Internal methods ───────────────────────────────────

func (g *Gateway) readPayload() (*GatewayPayload, error) {
	_, data, err := g.conn.Read(g.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	var payload GatewayPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// シーケンス番号を更新
	if payload.S != nil {
		g.seq.Store(*payload.S)
	}

	return &payload, nil
}

func (g *Gateway) sendPayload(op int, d interface{}) error {
	data, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	payload := GatewayPayload{
		Op: op,
		D:  json.RawMessage(data),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return g.conn.Write(g.ctx, websocket.MessageText, payloadJSON)
}

func (g *Gateway) readHello() error {
	payload, err := g.readPayload()
	if err != nil {
		return err
	}

	if payload.Op != OpcodeHello {
		return fmt.Errorf("expected Hello (op 10), got op %d", payload.Op)
	}

	var hello HelloData
	if err := json.Unmarshal(payload.D, &hello); err != nil {
		return fmt.Errorf("failed to unmarshal hello: %w", err)
	}

	g.heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	g.lastHeartbeatACK.Store(true)

	g.logger.Info("Received Hello", "heartbeat_interval_ms", hello.HeartbeatInterval)
	return nil
}

func (g *Gateway) sendIdentify() error {
	identify := IdentifyData{
		Token:   g.token,
		Intents: g.intents,
		Properties: IdentifyProperties{
			OS:      runtime.GOOS,
			Browser: "todoapp-discordbot",
			Device:  "todoapp-discordbot",
		},
	}

	g.logger.Info("Sending Identify")
	return g.sendPayload(OpcodeIdentify, identify)
}

func (g *Gateway) waitForReady() error {
	for {
		payload, err := g.readPayload()
		if err != nil {
			return err
		}

		if payload.Op == OpcodeDispatch && payload.T == "READY" {
			var ready ReadyData
			if err := json.Unmarshal(payload.D, &ready); err != nil {
				return fmt.Errorf("failed to unmarshal ready: %w", err)
			}

			g.sessionID = ready.SessionID
			g.resumeURL = ready.ResumeURL
			g.botUser = &ready.User

			g.logger.Info("Gateway READY",
				"user", ready.User.Username+"#"+ready.User.Discriminator,
				"session_id", ready.SessionID,
			)
			return nil
		}

		// 他のイベントは無視（READY前に来ることがある）
		g.logger.Debug("Received pre-ready event", "op", payload.Op, "t", payload.T)
	}
}

func (g *Gateway) heartbeatLoop() {
	ticker := time.NewTicker(g.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			if !g.lastHeartbeatACK.Load() {
				g.logger.Warn("No heartbeat ACK received, connection may be zombie")
			}

			g.lastHeartbeatACK.Store(false)
			seq := g.seq.Load()

			if err := g.sendPayload(OpcodeHeartbeat, seq); err != nil {
				g.logger.Error("Failed to send heartbeat", "error", err)
				return
			}

			g.logger.Debug("Sent heartbeat", "seq", seq)
		}
	}
}

func (g *Gateway) listenLoop() {
	defer close(g.done)

	for {
		payload, err := g.readPayload()
		if err != nil {
			select {
			case <-g.ctx.Done():
				g.logger.Info("Gateway listen loop stopped (context cancelled)")
				return
			default:
				g.logger.Error("Failed to read payload", "error", err)
				return
			}
		}

		switch payload.Op {
		case OpcodeDispatch:
			g.dispatchEvent(payload.T, payload.D)

		case OpcodeHeartbeat:
			// サーバーからのハートビート要求 → 即座に返す
			seq := g.seq.Load()
			if err := g.sendPayload(OpcodeHeartbeat, seq); err != nil {
				g.logger.Error("Failed to send heartbeat response", "error", err)
			}

		case OpcodeReconnect:
			g.logger.Info("Received Reconnect request")
			return

		case OpcodeInvalidSession:
			g.logger.Warn("Received Invalid Session")
			return

		case OpcodeHeartbeatACK:
			g.lastHeartbeatACK.Store(true)
			g.logger.Debug("Received heartbeat ACK")

		default:
			g.logger.Debug("Received unknown opcode", "op", payload.Op)
		}
	}
}

func (g *Gateway) dispatchEvent(eventName string, data json.RawMessage) {
	g.mu.RLock()
	handlers := g.handlers[eventName]
	g.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}
