package domain

import "context"

// ChannelRepository はチャンネル情報を取得するリポジトリインターフェース
type ChannelRepository interface {
	FindByName(ctx context.Context, name string) (*Channel, error)
}

// MessageRepository はメッセージを取得するリポジトリインターフェース
type MessageRepository interface {
	FindByChannel(ctx context.Context, channelID string, dateRange *DateRange) ([]*Message, error)
	FindThreadReplies(ctx context.Context, channelID string, threadTS string, dateRange *DateRange) ([]*Message, error)
}

// UserRepository はユーザー情報を取得するリポジトリインターフェース
type UserRepository interface {
	FindByIDs(ctx context.Context, userIDs []string) (map[string]*User, error)
	FindAll(ctx context.Context) (map[string]*User, error)
}
