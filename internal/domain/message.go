package domain

import "time"

// Message はSlackメッセージを表すドメインモデル
type Message struct {
	ID          string
	Text        string
	UserID      string
	ChannelID   string
	Timestamp   time.Time
	Reactions   []Reaction
	IsBot       bool
	ThreadTS    string // スレッドのタイムスタンプ（空文字列の場合は通常メッセージ）
}

// HasReactions はメッセージにリアクションがあるかどうかを返す
func (m *Message) HasReactions() bool {
	return len(m.Reactions) > 0
}

// TotalReactionCount はメッセージの総リアクション数を返す
func (m *Message) TotalReactionCount() int {
	total := 0
	for _, r := range m.Reactions {
		total += r.Count
	}
	return total
}

// IsThreadReply はこのメッセージがスレッドの返信かどうかを返す
func (m *Message) IsThreadReply() bool {
	return m.ThreadTS != "" && m.ThreadTS != m.ID
}

// IsThreadParent はこのメッセージがスレッドの親メッセージかどうかを返す
func (m *Message) IsThreadParent() bool {
	return m.ThreadTS != "" && m.ThreadTS == m.ID
}
