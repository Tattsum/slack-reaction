package slack

import (
	"context"
	"fmt"

	"github.com/Tattsum/slack-reaction/internal/domain"
	"github.com/slack-go/slack"
)

// ChannelRepository はSlack APIを使用してチャンネル情報を取得するリポジトリ
type ChannelRepository struct {
	client *slack.Client
}

// NewChannelRepository は新しいChannelRepositoryを作成する
func NewChannelRepository(client *slack.Client) *ChannelRepository {
	return &ChannelRepository{
		client: client,
	}
}

// FindByName はチャンネル名からチャンネルを検索する
func (r *ChannelRepository) FindByName(ctx context.Context, name string) (*domain.Channel, error) {
	conversations, cursor, err := r.client.GetConversations(&slack.GetConversationsParameters{
		ExcludeArchived: true,
		Limit:           1000,
	})
	if err != nil {
		return nil, fmt.Errorf("チャンネル一覧取得エラー: %w", err)
	}

	// 指定されたチャンネル名に一致するチャンネルを検索
	for _, conversation := range conversations {
		if conversation.Name == name {
			return &domain.Channel{
				ID:   conversation.ID,
				Name: conversation.Name,
			}, nil
		}
	}

	// カーソルがある場合、次のページを取得
	for cursor != "" {
		conversations, cursor, err = r.client.GetConversations(&slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           1000,
			Cursor:          cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("チャンネル一覧取得エラー: %w", err)
		}

		for _, conversation := range conversations {
			if conversation.Name == name {
				return &domain.Channel{
					ID:   conversation.ID,
					Name: conversation.Name,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("チャンネル '%s' が見つかりません", name)
}

// FindAll はすべてのチャンネルを取得する
func (r *ChannelRepository) FindAll(ctx context.Context) ([]*domain.Channel, error) {
	var allChannels []*domain.Channel
	cursor := ""
	
	for {
		conversations, nextCursor, err := r.client.GetConversations(&slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           1000,
			Cursor:          cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("チャンネル一覧取得エラー: %w", err)
		}

		for _, conversation := range conversations {
			allChannels = append(allChannels, &domain.Channel{
				ID:   conversation.ID,
				Name: conversation.Name,
			})
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return allChannels, nil
}
