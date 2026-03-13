package discord

import "encoding/json"

// ─── Gateway Opcodes ────────────────────────────────────

const (
	OpcodeDispatch            = 0
	OpcodeHeartbeat           = 1
	OpcodeIdentify            = 2
	OpcodePresenceUpdate      = 3
	OpcodeVoiceStateUpdate    = 4
	OpcodeResume              = 6
	OpcodeReconnect           = 7
	OpcodeRequestGuildMembers = 8
	OpcodeInvalidSession      = 9
	OpcodeHello               = 10
	OpcodeHeartbeatACK        = 11
)

// ─── Gateway payloads ───────────────────────────────────

// GatewayPayload はGateway WebSocketのメッセージ
type GatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  *int64          `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

// HelloData は opcode 10 (Hello) のデータ
type HelloData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// IdentifyData は opcode 2 (Identify) のデータ
type IdentifyData struct {
	Token      string             `json:"token"`
	Intents    int                `json:"intents"`
	Properties IdentifyProperties `json:"properties"`
}

// IdentifyProperties はIdentifyのプロパティ
type IdentifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

// ResumeData は opcode 6 (Resume) のデータ
type ResumeData struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
}

// ReadyData は READY イベントのデータ
type ReadyData struct {
	V         int    `json:"v"`
	User      User   `json:"user"`
	SessionID string `json:"session_id"`
	ResumeURL string `json:"resume_gateway_url"`
}

// ─── Discord API types ──────────────────────────────────

// User はDiscordユーザーを表す
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name,omitempty"`
	Bot           bool   `json:"bot,omitempty"`
}

// ─── Interaction types ──────────────────────────────────

// InteractionType 定数
const (
	InteractionTypePing               = 1
	InteractionTypeApplicationCommand = 2
	InteractionTypeMessageComponent   = 3
	InteractionTypeAutocomplete       = 4
	InteractionTypeModalSubmit        = 5
)

// InteractionCallbackType 定数
const (
	InteractionCallbackPong                     = 1
	InteractionCallbackChannelMessageWithSource = 4
	InteractionCallbackDeferredChannelMessage   = 5
	InteractionCallbackDeferredUpdateMessage    = 6
	InteractionCallbackUpdateMessage            = 7
)

// Interaction はDiscordインタラクションを表す
type Interaction struct {
	ID        string           `json:"id"`
	Type      int              `json:"type"`
	Token     string           `json:"token"`
	GuildID   string           `json:"guild_id,omitempty"`
	ChannelID string           `json:"channel_id,omitempty"`
	Data      *InteractionData `json:"data,omitempty"`
	AppID     string           `json:"application_id,omitempty"`
}

// InteractionData はインタラクションのデータ
type InteractionData struct {
	ID      string                   `json:"id"`
	Name    string                   `json:"name"`
	Options []*InteractionDataOption `json:"options,omitempty"`
}

// InteractionDataOption はインタラクションオプション
type InteractionDataOption struct {
	Name    string                   `json:"name"`
	Type    int                      `json:"type"`
	Value   interface{}              `json:"value,omitempty"`
	Options []*InteractionDataOption `json:"options,omitempty"`
}

// StringValue はオプションの文字列値を返す
func (o *InteractionDataOption) StringValue() string {
	if s, ok := o.Value.(string); ok {
		return s
	}
	return ""
}

// IntValue はオプションの整数値を返す
func (o *InteractionDataOption) IntValue() int64 {
	if f, ok := o.Value.(float64); ok {
		return int64(f)
	}
	return 0
}

// InteractionResponse はインタラクションレスポンス
type InteractionResponse struct {
	Type int                      `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// InteractionResponseData はインタラクションレスポンスのデータ
type InteractionResponseData struct {
	Content string          `json:"content,omitempty"`
	Embeds  []*MessageEmbed `json:"embeds,omitempty"`
}

// ─── Command types ──────────────────────────────────────

// ApplicationCommandOptionType 定数
const (
	OptionTypeSubCommand      = 1
	OptionTypeSubCommandGroup = 2
	OptionTypeString          = 3
	OptionTypeInteger         = 4
	OptionTypeBoolean         = 5
	OptionTypeUser            = 6
	OptionTypeChannel         = 7
	OptionTypeRole            = 8
	OptionTypeMentionable     = 9
	OptionTypeNumber          = 10
)

// ApplicationCommand はスラッシュコマンド定義
type ApplicationCommand struct {
	ID          string                      `json:"id,omitempty"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Options     []*ApplicationCommandOption `json:"options,omitempty"`
}

// ApplicationCommandOption はコマンドオプション定義
type ApplicationCommandOption struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Type        int                               `json:"type"`
	Required    bool                              `json:"required,omitempty"`
	Options     []*ApplicationCommandOption       `json:"options,omitempty"`
	Choices     []*ApplicationCommandOptionChoice `json:"choices,omitempty"`
}

// ApplicationCommandOptionChoice はオプションの選択肢
type ApplicationCommandOptionChoice struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// ─── Embed types ────────────────────────────────────────

// MessageEmbed はメッセージ埋め込み
type MessageEmbed struct {
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Color       int                  `json:"color,omitempty"`
	Fields      []*MessageEmbedField `json:"fields,omitempty"`
	Footer      *MessageEmbedFooter  `json:"footer,omitempty"`
}

// MessageEmbedField は埋め込みフィールド
type MessageEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// MessageEmbedFooter は埋め込みフッター
type MessageEmbedFooter struct {
	Text string `json:"text"`
}

// ─── Gateway Bot response ───────────────────────────────

// GatewayBotResponse は GET /gateway/bot のレスポンス
type GatewayBotResponse struct {
	URL    string `json:"url"`
	Shards int    `json:"shards"`
}
