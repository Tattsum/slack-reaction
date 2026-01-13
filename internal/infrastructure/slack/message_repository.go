package slack

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Tattsum/slack-reaction/internal/domain"
	"github.com/slack-go/slack"
)

// MessageRepository はSlack APIを使用してメッセージを取得するリポジトリ
type MessageRepository struct {
	client *slack.Client
}

// NewMessageRepository は新しいMessageRepositoryを作成する
func NewMessageRepository(client *slack.Client) *MessageRepository {
	return &MessageRepository{
		client: client,
	}
}

// FindByChannel はチャンネルのメッセージを取得する
func (r *MessageRepository) FindByChannel(ctx context.Context, channelID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	var oldest, latest string
	if dateRange != nil {
		if !dateRange.Start.IsZero() {
			oldest = fmt.Sprintf("%d", dateRange.Start.Unix())
		}
		if !dateRange.End.IsZero() {
			latest = fmt.Sprintf("%d", dateRange.End.Unix())
		}
	}

	params := slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Oldest:    oldest,
		Latest:    latest,
		Limit:     1000,
	}

	var messages []*domain.Message
	hasMore := true

	for hasMore {
		history, err := r.client.GetConversationHistoryContext(ctx, &params)
		if err != nil {
			return nil, fmt.Errorf("メッセージ取得エラー: %w", err)
		}

		for _, msg := range history.Messages {
			domainMsg := r.convertToDomainMessage(&msg, channelID)
			if domainMsg != nil {
				messages = append(messages, domainMsg)
			}
		}

		hasMore = history.HasMore
		if hasMore {
			params.Cursor = history.ResponseMetaData.NextCursor
		}
	}

	return messages, nil
}

// FindThreadReplies はスレッドの返信を取得する
func (r *MessageRepository) FindThreadReplies(ctx context.Context, channelID string, threadTS string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	var oldest, latest string
	if dateRange != nil {
		if !dateRange.Start.IsZero() {
			oldest = fmt.Sprintf("%d", dateRange.Start.Unix())
		}
		if !dateRange.End.IsZero() {
			latest = fmt.Sprintf("%d", dateRange.End.Unix())
		}
	}

	params := slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: threadTS,
		Oldest:    oldest,
		Latest:    latest,
		Limit:     1000,
	}

	const maxRetries = 3
	var messages []*domain.Message

	for retry := 0; retry < maxRetries; retry++ {
		replies, hasMore, _, err := r.client.GetConversationRepliesContext(ctx, &params)
		if err != nil {
			if isRateLimitError(err) {
				sleepTime := extractRetryAfter(err.Error())
				if sleepTime > 0 {
					time.Sleep(time.Duration(sleepTime) * time.Second)
					continue
				} else {
					time.Sleep(time.Duration(10+retry*5) * time.Second)
					continue
				}
			}
			return nil, fmt.Errorf("スレッドメッセージ取得エラー: %w", err)
		}

		// 最初のメッセージ（親メッセージ）をスキップ
		if len(replies) > 1 {
			for _, msg := range replies[1:] {
				domainMsg := r.convertToDomainMessage(&msg, channelID)
				if domainMsg != nil {
					// 日付範囲チェック
					if dateRange != nil && !dateRange.Contains(domainMsg.Timestamp) {
						continue
					}
					messages = append(messages, domainMsg)
				}
			}
		}

		if !hasMore {
			break
		}
	}

	return messages, nil
}

// convertToDomainMessage はSlackのMessageをドメインモデルに変換する
func (r *MessageRepository) convertToDomainMessage(msg *slack.Message, channelID string) *domain.Message {
	// ボットメッセージをスキップ
	if msg.SubType == "bot_message" || msg.BotID != "" {
		return nil
	}

	timestamp, _ := parseSlackTimestamp(msg.Timestamp)

	reactions := make([]domain.Reaction, 0, len(msg.Reactions))
	for _, reaction := range msg.Reactions {
		reactions = append(reactions, domain.Reaction{
			Name:  reaction.Name,
			Count: reaction.Count,
		})
	}

	return &domain.Message{
		ID:        msg.Timestamp,
		Text:      msg.Text,
		UserID:    msg.User,
		ChannelID: channelID,
		Timestamp: timestamp,
		Reactions: reactions,
		IsBot:     msg.SubType == "bot_message" || msg.BotID != "",
		ThreadTS:  msg.ThreadTimestamp,
	}
}

// parseSlackTimestamp はSlackのタイムスタンプ文字列をtime.Timeに変換する
func parseSlackTimestamp(ts string) (time.Time, error) {
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return time.Time{}, fmt.Errorf("無効なタイムスタンプ: %s", ts)
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("タイムスタンプ解析エラー: %w", err)
	}

	return time.Unix(sec, 0), nil
}

// isRateLimitError はレート制限エラーかチェック
func isRateLimitError(err error) bool {
	return strings.Contains(err.Error(), "rate limit exceeded")
}

// extractRetryAfter はエラーメッセージからretry-after時間を抽出
func extractRetryAfter(errMsg string) int {
	if strings.Contains(errMsg, "retry after") {
		parts := strings.Split(errMsg, "retry after ")
		if len(parts) > 1 {
			timeStr := strings.TrimSuffix(parts[1], "s")
			if retryAfter, err := strconv.Atoi(timeStr); err == nil {
				return retryAfter
			}
		}
	}
	return 0
}
