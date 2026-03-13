# discord-api-go

個人用途のために作成された、
Go 言語向けの Discord API クライアントライブラリです。REST API と Gateway WebSocket を統合サポートします。

## インストール

```bash
go get github.com/sikigasa/discord-api-go
```

## 概要

### モジュール構成

このライブラリは以下の主要コンポーネントで構成されます：

- **API** - Discord REST API クライアント
  - Gateway Bot URL 取得
  - グローバルコマンド登録
  - インタラクションレスポンス
- **Gateway** - Discord Gateway WebSocket 接続管理
  - WebSocket 接続・切断
  - イベントハンドラー登録
  - ハートビート管理
  - READY 状態確認
- **Client** - 統合クライアント
  - REST API と Gateway を一つのインターフェースで提供
  - インタラクション処理
  - コマンド登録

## 使用例

### 基本的な使い方

```go
package main

import (
 "context"
 "log/slog"
 "os"

 "github.com/sikigasa/discord-api-go"
)

func main() {
 // ロガーとクライアント初期化
 logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
 client := discord.NewClient(os.Getenv("DISCORD_BOT_TOKEN"), logger)

 // インタラクションハンドラー登録
 client.OnInteraction(func(inter *discord.Interaction) {
  if inter.Type == discord.InteractionTypeApplicationCommand {
   response := &discord.InteractionResponse{
    Type: discord.InteractionCallbackChannelMessageWithSource,
    Data: &discord.InteractionResponseData{
     Content: "Pong!",
    },
   }
   client.RespondToInteraction(inter.ID, inter.Token, response)
  }
 })

 // Gateway に接続
 if err := client.Open(); err != nil {
  logger.Error("Failed to open client", "error", err)
  return
 }
 defer client.Close()

 // コマンド登録例
 commands := []*discord.ApplicationCommand{
  {
   Name:        "ping",
   Description: "Ping コマンド",
  },
 }
 if err := client.RegisterCommands(commands); err != nil {
  logger.Error("Failed to register commands", "error", err)
  return
 }

 // イベント処理中...
 <-client.Gateway.Done()
}
```

## API 仕様

### Client

#### NewClient(token string, logger \*slog.Logger) \*Client

新しい統合クライアントを作成します。

**パラメータ:**

- `token` - Discord Bot Token
- `logger` - slog.Logger インスタンス

**戻り値:**

- 初期化された `Client` インスタンス

#### func (c \*Client) OnInteraction(handler func(\*Interaction))

インタラクションハンドラーを登録します。

#### func (c \*Client) Open() error

Gateway に接続し、イベント購読を開始します。

#### func (c \*Client) Close() error

Gateway 接続を閉じます。

#### func (c \*Client) BotUser() \*User

Bot 自身のユーザー情報を返します（接続後）。

#### func (c \*Client) RegisterCommands(commands [\]\*ApplicationCommand) error

スラッシュコマンドを一括登録します。

#### func (c \*Client) RespondToInteraction(interactionID, interactionToken string, response \*InteractionResponse) error

インタラクションに応答します。

### API

#### NewAPI(token string, logger \*slog.Logger) \*API

新しい Discord REST API クライアントを作成します。

#### func (a \*API) GetGatewayBot() (\*GatewayBotResponse, error)

Gateway Bot URL とシャード情報を取得します。

#### func (a \*API) BulkOverwriteGlobalCommands(appID string, commands [\]\*ApplicationCommand) ([\]\*ApplicationCommand, error)

グローバルコマンドを一括登録します。

#### func (a \*API) CreateInteractionResponse(interactionID, interactionToken string, response \*InteractionResponse) error

インタラクションに応答します。

### Gateway

#### NewGateway(token string, intents int, logger \*slog.Logger) \*Gateway

新しい Gateway 接続を作成します。

**パラメータ:**

- `token` - Discord Bot Token
- `intents` - Gateway Intents フラグ（通常は 0）
- `logger` - slog.Logger インスタンス

#### func (g \*Gateway) On(event string, handler func(json.RawMessage))

イベントハンドラーを登録します。

**対応イベント:**

- `READY` - 接続準備完了
- `INTERACTION_CREATE` - インタラクション受信

#### func (g \*Gateway) Connect(gatewayURL string) error

Gateway に接続します。

#### func (g \*Gateway) Close() error

Gateway 接続を切断します。

#### func (g \*Gateway) BotUser() \*User

Bot 自身のユーザー情報を返します。

#### func (g \*Gateway) Done() <-chan struct{}

接続が閉じられたときに閉じるチャネルを返します。

## 型定義

### Interaction

Discord インタラクション構造です。

```go
type Interaction struct {
 ID        string           // インタラクション ID
 Type      int              // InteractionType 定数
 Token     string           // インタラクショントークン
 GuildID   string           // ギルド ID（オプション）
 ChannelID string           // チャネル ID（オプション）
 Data      *InteractionData // インタラクションデータ
 AppID     string           // アプリケーション ID（オプション）
}
```

### InteractionResponse

インタラクションレスポンス構造です。

```go
type InteractionResponse struct {
 Type int                      // InteractionCallback* 定数
 Data *InteractionResponseData // レスポンスデータ
}
```

### ApplicationCommand

スラッシュコマンド定義です。

```go
type ApplicationCommand struct {
 ID          string                      // コマンド ID（取得時のみ）
 Name        string                      // コマンド名
 Description string                      // コマンド説明
 Options     []*ApplicationCommandOption // オプション（オプション）
}
```
