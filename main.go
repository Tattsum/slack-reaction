package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
)

type EmojiCount struct {
	Emoji string
	Count int
}

type MessageReaction struct {
	Text      string
	Reactions int
	Timestamp string
}

type UserStats struct {
	UserID   string
	UserName string
	Count    int
}

func main() {
	log.Println("=== Slack リアクション分析ツール 開始 ===")

	// コマンドライン引数の定義
	channelName := flag.String("channel", "", "分析対象のSlackチャンネル名（必須）")
	startDate := flag.String("start", "", "開始日時（YYYY-MM-DD形式、省略可）")
	endDate := flag.String("end", "", "終了日時（YYYY-MM-DD形式、省略可）")
	flag.Parse()

	log.Printf("設定: チャンネル=%s, 開始日=%s, 終了日=%s", *channelName, *startDate, *endDate)

	// チャンネル名が指定されていない場合はエラー
	if *channelName == "" {
		fmt.Println("エラー: チャンネル名を指定してください")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Slack APIトークンを環境変数から取得
	token := os.Getenv("SLACK_USER_TOKEN")
	if token == "" {
		log.Fatal("環境変数 SLACK_USER_TOKEN が設定されていません")
	}
	log.Printf("トークンを取得しました (長さ: %d文字)", len(token))

	// Slackクライアントの初期化
	api := slack.New(token)

	// 日付範囲の設定
	var oldest, latest string
	if *startDate != "" {
		startTime, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			log.Fatalf("開始日時の形式が無効です: %v", err)
		}
		oldest = fmt.Sprintf("%d", startTime.Unix())
	}
	if *endDate != "" {
		endTime, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("終了日時の形式が無効です: %v", err)
		}
		// 指定日の終わり（23:59:59）を設定
		endTime = endTime.Add(24*time.Hour - time.Second)
		latest = fmt.Sprintf("%d", endTime.Unix())
	}

	// チャンネルIDの取得
	channelID, err := getChannelID(api, *channelName)
	if err != nil {
		log.Fatalf("チャンネルの取得に失敗しました: %v", err)
	}

	// メッセージの取得と分析
	emojiStats, messageStats, userStats, err := analyzeChannel(api, channelID, oldest, latest)
	if err != nil {
		log.Fatalf("チャンネルの分析に失敗しました: %v", err)
	}

	// 結果の表示
	fmt.Println("\n===== 最も使用されたスタンプ TOP3 =====")
	for i, emoji := range emojiStats {
		if i >= 3 {
			break
		}
		fmt.Printf("%d位: :%s: - %d回\n", i+1, emoji.Emoji, emoji.Count)
	}

	fmt.Println("\n===== 最もリアクションがついたメッセージ TOP3 =====")
	for i, msg := range messageStats {
		if i >= 3 {
			break
		}
		// メッセージが長い場合は省略
		displayText := msg.Text
		if len(displayText) > 100 {
			displayText = displayText[:97] + "..."
		}
		fmt.Printf("%d位: %s\nリアクション数: %d\n\n", i+1, displayText, msg.Reactions)
	}

	fmt.Println("\n===== 最も投稿数が多いユーザー TOP10 =====")
	for i, user := range userStats {
		if i >= 10 {
			break
		}
		fmt.Printf("%d位: %s - %d投稿\n", i+1, user.UserName, user.Count)
	}
}

// チャンネル名からチャンネルIDを取得する関数
func getChannelID(api *slack.Client, channelName string) (string, error) {
	log.Printf("チャンネル一覧を取得中...")
	// チャンネルの一覧を取得
	conversations, cursor, err := api.GetConversations(&slack.GetConversationsParameters{
		ExcludeArchived: true,
		Limit:           1000,
	})
	if err != nil {
		log.Printf("チャンネル一覧取得エラー: %v", err)
		return "", err
	}

	log.Printf("取得したチャンネル数: %d", len(conversations))

	// 指定されたチャンネル名に一致するチャンネルを検索
	for _, conversation := range conversations {
		if conversation.Name == channelName {
			log.Printf("チャンネルを発見: %s (ID: %s)", channelName, conversation.ID)
			return conversation.ID, nil
		}
	}

	// カーソルがある場合、次のページを取得
	for cursor != "" {
		conversations, cursor, err = api.GetConversations(&slack.GetConversationsParameters{
			ExcludeArchived: true,
			Limit:           1000,
			Cursor:          cursor,
		})
		if err != nil {
			return "", err
		}

		for _, conversation := range conversations {
			if conversation.Name == channelName {
				return conversation.ID, nil
			}
		}
	}

	return "", fmt.Errorf("チャンネル '%s' が見つかりません", channelName)
}

// チャンネルのメッセージとリアクションを分析する関数（スレッド対応）
func analyzeChannel(api *slack.Client, channelID, oldest, latest string) ([]EmojiCount, []MessageReaction, []UserStats, error) {
	emojiCount := make(map[string]int)
	messageReactions := make([]MessageReaction, 0, 1000)
	userMessageCount := make(map[string]int)
	userIDs := make(map[string]bool)
	totalMessages := 0
	totalThreads := 0

	// メッセージの取得パラメータ
	params := slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Oldest:    oldest,
		Latest:    latest,
		Limit:     1000,
	}

	ctx := context.Background()
	hasMore := true
	pageCount := 0
	for hasMore {
		pageCount++
		log.Printf("メッセージ取得中... (ページ %d)", pageCount)

		// メッセージの取得
		history, err := api.GetConversationHistoryContext(ctx, &params)
		if err != nil {
			log.Printf("メッセージ取得エラー: %v", err)
			return nil, nil, nil, err
		}

		// 各メッセージを処理
		log.Printf("取得したメッセージ数: %d", len(history.Messages))
		messageCount := 0
		for i, msg := range history.Messages {
			messageCount++
			if messageCount%50 == 0 || i == len(history.Messages)-1 {
				log.Printf("メッセージ処理進捗: %d/%d (ページ %d)", messageCount, len(history.Messages), pageCount)
			}
			// ボットメッセージをスキップ
			if msg.SubType == "bot_message" || msg.BotID != "" {
				continue
			}

			// メインメッセージを処理
			processMessage(&msg, emojiCount, &messageReactions, userMessageCount, userIDs)
			totalMessages++

			if totalMessages%100 == 0 {
				log.Printf("累積メッセージ処理数: %d", totalMessages)
			}

			// スレッドがある場合はスレッドメッセージも処理（並行処理でパフォーマンス向上）
			if msg.ThreadTimestamp != "" && msg.ThreadTimestamp == msg.Timestamp {
				// このメッセージがスレッドの親メッセージの場合
				totalThreads++
				log.Printf("スレッド処理中... (#%d: %s)", totalThreads, msg.Timestamp)
				// レート制限を避けるため少し待機してからスレッド処理
				time.Sleep(100 * time.Millisecond)

				err := processThreadMessages(api, ctx, channelID, msg.Timestamp, oldest, latest, emojiCount, &messageReactions, userMessageCount, userIDs)
				if err != nil {
					log.Printf("スレッドメッセージの処理でエラーが発生しました: %v", err)
				} else {
					log.Printf("スレッド処理完了: #%d", totalThreads)
				}
			}
		}

		// 次のページがあるか確認
		hasMore = history.HasMore
		if hasMore {
			params.Cursor = history.ResponseMetaData.NextCursor
			log.Printf("次のページへ進みます... (カーソル: %.20s...)", params.Cursor)
		} else {
			log.Printf("全ページの処理が完了しました")
		}
	}

	// ユーザー名を一括取得
	userNames := fetchUserNames(api, userIDs)

	// 絵文字の使用回数でソート
	emojiStats := make([]EmojiCount, 0, len(emojiCount))
	for emoji, count := range emojiCount {
		emojiStats = append(emojiStats, EmojiCount{Emoji: emoji, Count: count})
	}
	sort.Slice(emojiStats, func(i, j int) bool {
		return emojiStats[i].Count > emojiStats[j].Count
	})

	// メッセージをリアクション数でソート
	sort.Slice(messageReactions, func(i, j int) bool {
		return messageReactions[i].Reactions > messageReactions[j].Reactions
	})

	// ユーザー統計を作成してソート
	userStats := make([]UserStats, 0, len(userMessageCount))
	for userID, count := range userMessageCount {
		userName := userNames[userID]
		if userName == "" {
			userName = userID
		}
		userStats = append(userStats, UserStats{
			UserID:   userID,
			UserName: userName,
			Count:    count,
		})
	}
	sort.Slice(userStats, func(i, j int) bool {
		return userStats[i].Count > userStats[j].Count
	})

	return emojiStats, messageReactions, userStats, nil
}

// 個別メッセージを処理する関数
func processMessage(msg *slack.Message, emojiCount map[string]int, messageReactions *[]MessageReaction, userMessageCount map[string]int, userIDs map[string]bool) {
	// ユーザーメッセージ数をカウント
	if msg.User != "" {
		userMessageCount[msg.User]++
		userIDs[msg.User] = true
	}

	totalReactions := 0

	// リアクションの集計
	for _, reaction := range msg.Reactions {
		emojiCount[reaction.Name] += reaction.Count
		totalReactions += reaction.Count
	}

	// メッセージとリアクション数を記録
	if totalReactions > 0 {
		*messageReactions = append(*messageReactions, MessageReaction{
			Text:      msg.Text,
			Reactions: totalReactions,
			Timestamp: msg.Timestamp,
		})
	}
}

// スレッドメッセージを処理する関数（レート制限対応）
func processThreadMessages(api *slack.Client, ctx context.Context, channelID, threadTimestamp, oldest, latest string, emojiCount map[string]int, messageReactions *[]MessageReaction, userMessageCount map[string]int, userIDs map[string]bool) error {
	const maxRetries = 3

	for retry := 0; retry < maxRetries; retry++ {
		params := slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: threadTimestamp,
			Oldest:    oldest,
			Latest:    latest,
			Limit:     1000,
		}

		replies, hasMore, _, err := api.GetConversationRepliesContext(ctx, &params)
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
			return err
		}

		// 最初のメッセージ（親メッセージ）をスキップ
		messages := replies
		if len(messages) > 1 {
			messages = messages[1:]
		}

		for _, msg := range messages {
			// ボットメッセージをスキップ
			if msg.SubType == "bot_message" || msg.BotID != "" {
				continue
			}

			// 日付範囲チェック
			if !isMessageInDateRange(&msg, oldest, latest) {
				continue
			}

			// スレッド内メッセージを処理
			processMessage(&msg, emojiCount, messageReactions, userMessageCount, userIDs)
		}

		// スレッドは通常ページネーションが少ないので1回で完了することが多い
		if !hasMore {
			break
		}
	}

	return nil
}

// レート制限エラーかチェック
func isRateLimitError(err error) bool {
	return strings.Contains(err.Error(), "rate limit exceeded")
}

// エラーメッセージからretry-after時間を抽出
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

// メッセージが指定された日付範囲内かチェックする関数
func isMessageInDateRange(msg *slack.Message, oldest, latest string) bool {
	if oldest != "" {
		msgTime, err := time.Parse("1504840306.000009", msg.Timestamp)
		if err == nil {
			oldestTime, err := time.Parse("1504840306", oldest)
			if err == nil && msgTime.Before(oldestTime) {
				return false
			}
		}
	}

	if latest != "" {
		msgTime, err := time.Parse("1504840306.000009", msg.Timestamp)
		if err == nil {
			latestTime, err := time.Parse("1504840306", latest)
			if err == nil && msgTime.After(latestTime) {
				return false
			}
		}
	}

	return true
}

// ユーザー名を一括で取得する最適化された関数
func fetchUserNames(api *slack.Client, userIDs map[string]bool) map[string]string {
	userNames := make(map[string]string)

	// 全ユーザーリストを一度だけ取得
	users, err := api.GetUsers()
	if err != nil {
		// エラーの場合は個別取得にフォールバック
		return fetchUserNamesIndividual(api, userIDs)
	}

	// ユーザーマップを作成
	userMap := make(map[string]*slack.User)
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	// 必要なユーザーの名前を抽出
	for userID := range userIDs {
		if user, exists := userMap[userID]; exists {
			if user.Profile.DisplayName != "" {
				userNames[userID] = user.Profile.DisplayName
			} else if user.RealName != "" {
				userNames[userID] = user.RealName
			} else if user.Name != "" {
				userNames[userID] = user.Name
			} else {
				userNames[userID] = userID
			}
		} else {
			userNames[userID] = userID
		}
	}

	return userNames
}

// 個別にユーザー情報を取得する関数（フォールバック用）
func fetchUserNamesIndividual(api *slack.Client, userIDs map[string]bool) map[string]string {
	userNames := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var processedCount int

	log.Printf("個別ユーザー情報取得開始 (対象: %d人)...", len(userIDs))
	// 並行処理でユーザー情報を取得（レート制限を考慮して制限）
	semaphore := make(chan struct{}, 10)

	for userID := range userIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			userInfo, err := api.GetUserInfo(id)

			mu.Lock()
			processedCount++
			if processedCount%10 == 0 {
				log.Printf("ユーザー情報取得進捗: %d/%d", processedCount, len(userIDs))
			}
			mu.Unlock()

			if err == nil && userInfo != nil {
				mu.Lock()
				if userInfo.Profile.DisplayName != "" {
					userNames[id] = userInfo.Profile.DisplayName
				} else if userInfo.RealName != "" {
					userNames[id] = userInfo.RealName
				} else if userInfo.Name != "" {
					userNames[id] = userInfo.Name
				} else {
					userNames[id] = id
				}
				mu.Unlock()
			} else {
				mu.Lock()
				userNames[id] = id
				mu.Unlock()
			}
		}(userID)
	}

	wg.Wait()
	return userNames
}
