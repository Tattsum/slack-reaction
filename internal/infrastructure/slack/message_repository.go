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
	pageCount := 0
	const maxRetries = 3

	for hasMore {
		pageCount++
		var history *slack.GetConversationHistoryResponse
		var err error

		// レート制限対応のリトライループ
		for retry := 0; retry < maxRetries; retry++ {
			history, err = r.client.GetConversationHistoryContext(ctx, &params)
			if err != nil {
				// not_in_channelエラーの場合は、より分かりやすいメッセージを表示
				if strings.Contains(err.Error(), "not_in_channel") {
					return nil, fmt.Errorf("チャンネル '%s' に参加していません。Slackでこのチャンネルに参加してから再度実行してください", channelID)
				}
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
				// レート制限以外のエラーは即座に返す
				if retry == maxRetries-1 {
					return nil, fmt.Errorf("メッセージ取得エラー: %w", err)
				}
			} else {
				// 成功したらループを抜ける
				break
			}
		}

		if err != nil {
			return nil, fmt.Errorf("メッセージ取得エラー: %w", err)
		}

		for _, msg := range history.Messages {
			domainMsg := r.convertToDomainMessage(&msg, channelID)
			if domainMsg != nil {
				messages = append(messages, domainMsg)
			}
		}

		fmt.Printf("メッセージ取得中... (ページ %d, 累計: %d件)\n", pageCount, len(messages))

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
			// not_in_channelエラーの場合は、より分かりやすいメッセージを表示
			if strings.Contains(err.Error(), "not_in_channel") {
				return nil, fmt.Errorf("チャンネル '%s' に参加していません。Slackでこのチャンネルに参加してから再度実行してください", channelID)
			}
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

// findByChannelSilentWithRetry はfindByChannelSilentをレート制限対応で呼び出す
func (r *MessageRepository) findByChannelSilentWithRetry(ctx context.Context, channelID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	const maxRetries = 3
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		messages, err := r.findByChannelSilent(ctx, channelID, dateRange)
		if err == nil {
			return messages, nil
		}

		lastErr = err

		// レート制限エラーの場合のみリトライ
		if isRateLimitError(err) {
			sleepTime := extractRetryAfter(err.Error())
			if sleepTime > 0 {
				time.Sleep(time.Duration(sleepTime) * time.Second)
			} else {
				time.Sleep(time.Duration(10+retry*5) * time.Second)
			}
			continue
		}

		// レート制限以外のエラーは即座に返す
		return nil, err
	}

	// 最大リトライ回数に達した場合は最後のエラーを返す
	return nil, lastErr
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
// エラーメッセージの形式: "slack rate limit exceeded, retry after 10s"
func extractRetryAfter(errMsg string) int {
	// "retry after" で分割
	if !strings.Contains(errMsg, "retry after") {
		return 0
	}

	parts := strings.Split(errMsg, "retry after ")
	if len(parts) < 2 {
		return 0
	}

	// parts[1] は "10s" のような形式
	timeStr := strings.TrimSpace(parts[1])
	
	// "s" を削除
	timeStr = strings.TrimSuffix(timeStr, "s")
	timeStr = strings.TrimSpace(timeStr)

	// 数値に変換
	if retryAfter, err := strconv.Atoi(timeStr); err == nil {
		return retryAfter
	}

	return 0
}

// FindByUser は指定されたユーザーIDのメッセージを全チャンネルから取得する
// まずSearch APIを試し、失敗した場合は既存の方法（全チャンネル横断）にフォールバック
func (r *MessageRepository) FindByUser(ctx context.Context, userID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	// Search APIを試す
	messages, err := r.findByUserWithSearchAPI(ctx, userID, dateRange)
	if err == nil {
		return messages, nil
	}

	// Search APIが使えない場合は、既存の方法にフォールバック
	fmt.Printf("Search APIが利用できないため、全チャンネル横断方式で検索します... (エラー: %v)\n", err)
	return r.findByUserFallback(ctx, userID, dateRange)
}

// findByUserWithSearchAPI はSearch APIを使用してユーザーのメッセージを検索する
func (r *MessageRepository) findByUserWithSearchAPI(ctx context.Context, userID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	// 検索クエリを構築: "from:userID"
	query := fmt.Sprintf("from:<@%s>", userID)

	// 日付範囲がある場合はクエリに追加
	if dateRange != nil {
		if !dateRange.Start.IsZero() {
			query += fmt.Sprintf(" after:%s", dateRange.Start.Format("2006-01-02"))
		}
		if !dateRange.End.IsZero() {
			query += fmt.Sprintf(" before:%s", dateRange.End.Format("2006-01-02"))
		}
	}

	var allMessages []*domain.Message
	page := 1
	const maxRetries = 3

	for {
	searchParams := slack.NewSearchParameters()
	searchParams.Count = 100 // 最大100件/ページ
	searchParams.Page = page

		var searchResults *slack.SearchMessages
		var err error

		// レート制限対応のリトライループ
		for retry := 0; retry < maxRetries; retry++ {
			searchResults, err = r.client.SearchMessages(query, searchParams)
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
				// レート制限以外のエラーは即座に返す
				if retry == maxRetries-1 {
					return nil, fmt.Errorf("Search APIエラー: %w", err)
				}
			} else {
				// 成功したらループを抜ける
				break
			}
		}

		if err != nil {
			return nil, fmt.Errorf("Search APIエラー: %w", err)
		}

		// 検索結果をドメインモデルに変換
		for _, match := range searchResults.Matches {
			msg := r.convertSearchResultToDomainMessage(&match, dateRange)
			if msg != nil && msg.UserID == userID {
				allMessages = append(allMessages, msg)
			}
		}

		fmt.Printf("Search API: ページ %d 処理完了 (累計: %d件)\n", page, len(allMessages))

		// 次のページがあるかチェック
		if page >= searchResults.Paging.Pages {
			break
		}
		page++
	}

	fmt.Printf("Search API検索完了: 合計 %d件のメッセージが見つかりました\n", len(allMessages))

	// リアクション情報とスレッド情報を補完するため、チャンネルごとにメッセージを取得
	// チャンネルIDのセットを作成
	channelIDs := make(map[string]bool)
	for _, msg := range allMessages {
		if msg.ChannelID != "" {
			channelIDs[msg.ChannelID] = true
		}
	}

	// 各チャンネルからメッセージを取得してリアクション情報を補完
	fmt.Printf("リアクション情報を補完中... (%dチャンネル)\n", len(channelIDs))
	channelMsgMap := make(map[string]map[string]*domain.Message) // channelID -> messageID -> message

	for channelID := range channelIDs {
		channelMessages, err := r.findByChannelSilent(ctx, channelID, dateRange)
		if err != nil {
			continue
		}
		msgMap := make(map[string]*domain.Message)
		for _, chMsg := range channelMessages {
			msgMap[chMsg.ID] = chMsg
		}
		channelMsgMap[channelID] = msgMap
	}

	// リアクション情報とスレッド情報を補完
	for _, msg := range allMessages {
		if msgMap, exists := channelMsgMap[msg.ChannelID]; exists {
			if fullMsg, found := msgMap[msg.ID]; found {
				msg.Reactions = fullMsg.Reactions
				msg.ThreadTS = fullMsg.ThreadTS

				// スレッドの親メッセージの場合は、スレッド返信も取得
				if msg.IsThreadParent() {
					replies, err := r.FindThreadReplies(ctx, msg.ChannelID, msg.ThreadTS, dateRange)
					if err == nil {
						// ユーザーのスレッド返信のみを追加
						for _, reply := range replies {
							if reply.UserID == userID {
								allMessages = append(allMessages, reply)
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("リアクション情報の補完完了: 合計 %d件のメッセージ\n", len(allMessages))
	return allMessages, nil
}

// convertSearchResultToDomainMessage はSearch APIの結果をドメインモデルに変換する
func (r *MessageRepository) convertSearchResultToDomainMessage(match *slack.SearchMessage, dateRange *domain.DateRange) *domain.Message {
	// タイムスタンプを解析
	timestamp, err := parseSlackTimestamp(match.Timestamp)
	if err != nil {
		return nil
	}

	// 日付範囲チェック
	if dateRange != nil && !dateRange.Contains(timestamp) {
		return nil
	}

	// リアクションを取得（Search APIの結果にはリアクション情報が含まれない可能性があるため、空スライス）
	reactions := make([]domain.Reaction, 0)

	// チャンネルIDを取得（Search APIの結果から）
	channelID := match.Channel.ID

	// スレッド情報を取得（Search APIの結果からはスレッド情報が直接取得できないため、後で確認する必要がある）
	// タイムスタンプをIDとして使用
	threadTS := match.Timestamp

	return &domain.Message{
		ID:        match.Timestamp,
		Text:      match.Text,
		UserID:    match.User,
		ChannelID: channelID,
		Timestamp: timestamp,
		Reactions: reactions,
		IsBot:     false, // Search APIの結果からは判定できないため、falseとする
		ThreadTS:  threadTS,
	}
}

// findByUserFallback は既存の方法（全チャンネル横断）でユーザーのメッセージを検索する
func (r *MessageRepository) findByUserFallback(ctx context.Context, userID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	// 全チャンネルを取得
	channelRepo := NewChannelRepository(r.client)
	channels, err := channelRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("チャンネル一覧取得エラー: %w", err)
	}

	var allMessages []*domain.Message
	channelCount := 0
	totalChannels := len(channels)

	fmt.Printf("全%dチャンネルからメッセージを検索します...\n", totalChannels)

	for _, channel := range channels {
		channelCount++

		// 進捗表示（10チャンネルごと、または最後のチャンネル）
		if channelCount%10 == 0 || channelCount == totalChannels {
			fmt.Printf("進捗: %d/%dチャンネル処理完了 (見つかったメッセージ: %d件)\n", channelCount, totalChannels, len(allMessages))
		}

		// チャンネルのメッセージを取得（進捗表示を抑制、レート制限対応）
		messages, err := r.findByChannelSilentWithRetry(ctx, channel.ID, dateRange)
		if err != nil {
			// チャンネルに参加していない場合はスキップ
			if strings.Contains(err.Error(), "参加していません") {
				continue
			}
			// レート制限エラーは既にリトライ済みなので、スキップ
			if isRateLimitError(err) {
				// レート制限エラーは既にリトライ済みなので、スキップ（警告は出さない）
				continue
			}
			// その他のエラーもログに記録するが、処理は続行
			if channelCount%10 == 0 || channelCount == totalChannels {
				fmt.Printf("警告: チャンネル '%s' のメッセージ取得エラー: %v\n", channel.Name, err)
			}
			continue
		}

		// 指定されたユーザーのメッセージのみをフィルタリング
		userMessageCount := 0
		for _, msg := range messages {
			if msg.UserID == userID {
				allMessages = append(allMessages, msg)
				userMessageCount++
			}
		}

		// スレッドの返信も取得（ユーザーのメッセージがある場合のみ）
		if userMessageCount > 0 {
			for _, msg := range messages {
				if msg.IsThreadParent() && msg.UserID == userID {
					replies, err := r.FindThreadReplies(ctx, channel.ID, msg.ThreadTS, dateRange)
					if err != nil {
						continue
					}
					// 指定されたユーザーのスレッド返信のみをフィルタリング
					for _, reply := range replies {
						if reply.UserID == userID {
							allMessages = append(allMessages, reply)
						}
					}
				}
			}
		}
	}

	fmt.Printf("メッセージ検索完了: 合計 %d件のメッセージが見つかりました\n", len(allMessages))
	return allMessages, nil
}

// findByChannelSilent はFindByChannelと同じだが、進捗表示を抑制する
func (r *MessageRepository) findByChannelSilent(ctx context.Context, channelID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
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
	const maxRetries = 3

	for hasMore {
		var history *slack.GetConversationHistoryResponse
		var err error

		// レート制限対応のリトライループ
		for retry := 0; retry < maxRetries; retry++ {
			history, err = r.client.GetConversationHistoryContext(ctx, &params)
			if err != nil {
				// not_in_channelエラーの場合は、より分かりやすいメッセージを表示
				if strings.Contains(err.Error(), "not_in_channel") {
					return nil, fmt.Errorf("チャンネル '%s' に参加していません。Slackでこのチャンネルに参加してから再度実行してください", channelID)
				}
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
				// レート制限以外のエラーは即座に返す
				if retry == maxRetries-1 {
					return nil, fmt.Errorf("メッセージ取得エラー: %w", err)
				}
			} else {
				// 成功したらループを抜ける
				break
			}
		}

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
