package service

import (
	"context"
	"sort"

	"github.com/Tattsum/slack-reaction/internal/domain"
)

// Analyzer はチャンネルのメッセージとリアクションを分析するサービス
type Analyzer struct {
	messageRepo domain.MessageRepository
	userRepo    domain.UserRepository
}

// NewAnalyzer は新しいAnalyzerサービスを作成する
func NewAnalyzer(messageRepo domain.MessageRepository, userRepo domain.UserRepository) *Analyzer {
	return &Analyzer{
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

// AnalyzeChannel はチャンネルのメッセージとリアクションを分析する
func (a *Analyzer) AnalyzeChannel(ctx context.Context, channelID string, dateRange *domain.DateRange) (*AnalysisResult, error) {
	// メッセージを取得
	messages, err := a.messageRepo.FindByChannel(ctx, channelID, dateRange)
	if err != nil {
		return nil, err
	}

	// スレッドの返信も取得
	for _, msg := range messages {
		if msg.IsThreadParent() {
			replies, err := a.messageRepo.FindThreadReplies(ctx, channelID, msg.ThreadTS, dateRange)
			if err != nil {
				// エラーはログに記録するが、処理は続行
				continue
			}
			messages = append(messages, replies...)
		}
	}

	// 分析結果を集計
	result := a.aggregate(messages)

	// ユーザー名を取得
	userIDs := make([]string, 0, len(result.UserMessageCount))
	for userID := range result.UserMessageCount {
		userIDs = append(userIDs, userID)
	}

	users, err := a.userRepo.FindByIDs(ctx, userIDs)
	if err != nil {
		// エラーの場合は空のマップを使用
		users = make(map[string]*domain.User)
	}

	// ユーザー統計を作成
	result.UserStats = a.buildUserStats(result.UserMessageCount, users)

	return result, nil
}

// AnalysisResult は分析結果を表す
type AnalysisResult struct {
	EmojiStats      []domain.EmojiCount
	MessageStats    []domain.MessageReaction
	UserStats       []domain.UserStats
	UserMessageCount map[string]int
}

// aggregate はメッセージから統計情報を集計する
func (a *Analyzer) aggregate(messages []*domain.Message) *AnalysisResult {
	emojiCount := make(map[string]int)
	messageReactions := make([]domain.MessageReaction, 0)
	userMessageCount := make(map[string]int)

	for _, msg := range messages {
		// ボットメッセージをスキップ
		if msg.IsBot {
			continue
		}

		// ユーザーメッセージ数をカウント
		if msg.UserID != "" {
			userMessageCount[msg.UserID]++
		}

		// リアクションを集計
		totalReactions := msg.TotalReactionCount()
		for _, reaction := range msg.Reactions {
			emojiCount[reaction.Name] += reaction.Count
		}

		// メッセージとリアクション数を記録
		if totalReactions > 0 {
			messageReactions = append(messageReactions, domain.MessageReaction{
				Text:      msg.Text,
				Reactions: totalReactions,
				Timestamp: msg.Timestamp.Format("20060102.150405"),
			})
		}
	}

	// 絵文字の使用回数でソート
	emojiStats := make([]domain.EmojiCount, 0, len(emojiCount))
	for emoji, count := range emojiCount {
		emojiStats = append(emojiStats, domain.EmojiCount{Emoji: emoji, Count: count})
	}
	sort.Slice(emojiStats, func(i, j int) bool {
		return emojiStats[i].Count > emojiStats[j].Count
	})

	// メッセージをリアクション数でソート
	sort.Slice(messageReactions, func(i, j int) bool {
		return messageReactions[i].Reactions > messageReactions[j].Reactions
	})

	return &AnalysisResult{
		EmojiStats:      emojiStats,
		MessageStats:    messageReactions,
		UserMessageCount: userMessageCount,
	}
}

// buildUserStats はユーザー統計を作成する
func (a *Analyzer) buildUserStats(userMessageCount map[string]int, users map[string]*domain.User) []domain.UserStats {
	userStats := make([]domain.UserStats, 0, len(userMessageCount))
	for userID, count := range userMessageCount {
		user := users[userID]
		userName := userID
		if user != nil {
			userName = user.GetDisplayName()
		}
		userStats = append(userStats, domain.UserStats{
			UserID:   userID,
			UserName: userName,
			Count:    count,
		})
	}

	// 投稿数でソート
	sort.Slice(userStats, func(i, j int) bool {
		return userStats[i].Count > userStats[j].Count
	})

	return userStats
}
