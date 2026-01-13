package domain

// Reaction はSlackのリアクション（絵文字）を表すドメインモデル
type Reaction struct {
	Name  string // 絵文字名（例: "thumbsup", "smile"）
	Count int    // リアクション数
}

// EmojiCount は絵文字の使用回数を集計するためのドメインモデル
type EmojiCount struct {
	Emoji string
	Count int
}

// MessageReaction はメッセージとそのリアクション数を表すドメインモデル
type MessageReaction struct {
	Text      string
	Reactions int
	Timestamp string
}
