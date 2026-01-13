package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tattsum/slack-reaction/internal/domain"
)

// mockMessageRepository はMessageRepositoryのモック実装
type mockMessageRepository struct {
	messages []*domain.Message
	err      error
}

func (m *mockMessageRepository) FindByChannel(ctx context.Context, channelID string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.messages, nil
}

func (m *mockMessageRepository) FindThreadReplies(ctx context.Context, channelID string, threadTS string, dateRange *domain.DateRange) ([]*domain.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	// スレッドの返信をフィルタリング
	replies := make([]*domain.Message, 0)
	for _, msg := range m.messages {
		if msg.ThreadTS == threadTS && msg.IsThreadReply() {
			replies = append(replies, msg)
		}
	}
	return replies, nil
}

// mockUserRepository はUserRepositoryのモック実装
type mockUserRepository struct {
	users map[string]*domain.User
	err   error
}

func (m *mockUserRepository) FindByIDs(ctx context.Context, userIDs []string) (map[string]*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[string]*domain.User)
	for _, userID := range userIDs {
		if user, exists := m.users[userID]; exists {
			result[userID] = user
		} else {
			result[userID] = &domain.User{ID: userID, Name: userID}
		}
	}
	return result, nil
}

func (m *mockUserRepository) FindAll(ctx context.Context) (map[string]*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func TestAnalyzer_AnalyzeChannel(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		messages      []*domain.Message
		users         map[string]*domain.User
		wantEmojiTop  int
		wantMsgTop    int
		wantUserTop   int
		wantThreadTop int
	}{
		{
			name: "基本的な分析",
			messages: []*domain.Message{
				{
					ID:        "1",
					Text:      "メッセージ1",
					UserID:    "U1",
					ChannelID: "C1",
					Timestamp: now,
					Reactions: []domain.Reaction{
						{Name: "thumbsup", Count: 5},
						{Name: "smile", Count: 3},
					},
					IsBot: false,
				},
				{
					ID:        "2",
					Text:      "メッセージ2",
					UserID:    "U2",
					ChannelID: "C1",
					Timestamp: now,
					Reactions: []domain.Reaction{
						{Name: "thumbsup", Count: 10},
					},
					IsBot: false,
				},
			},
			users: map[string]*domain.User{
				"U1": {ID: "U1", Name: "User1", DisplayName: "ユーザー1"},
				"U2": {ID: "U2", Name: "User2", DisplayName: "ユーザー2"},
			},
			wantEmojiTop:  1, // thumbsupが15回で最多
			wantMsgTop:    1, // メッセージ2が10回で最多
			wantUserTop:   2, // 両方のユーザーが1回ずつ
			wantThreadTop: 0, // スレッドなし
		},
		{
			name: "ボットメッセージをスキップ",
			messages: []*domain.Message{
				{
					ID:        "1",
					Text:      "ボットメッセージ",
					UserID:    "B1",
					ChannelID: "C1",
					Timestamp: now,
					IsBot:     true,
				},
				{
					ID:        "2",
					Text:      "通常メッセージ",
					UserID:    "U1",
					ChannelID: "C1",
					Timestamp: now,
					Reactions: []domain.Reaction{
						{Name: "thumbsup", Count: 5},
					},
					IsBot: false,
				},
			},
			users: map[string]*domain.User{
				"U1": {ID: "U1", Name: "User1"},
			},
			wantEmojiTop:  1,
			wantMsgTop:    1,
			wantUserTop:   1, // ボットはカウントされない
			wantThreadTop: 0, // スレッドなし
		},
		{
			name: "スレッドのコメント数ランキング",
			messages: []*domain.Message{
				{
					ID:        "1",
					Text:      "スレッド親1",
					UserID:    "U1",
					ChannelID: "C1",
					Timestamp: now,
					ThreadTS:  "1", // スレッドの親
					IsBot:     false,
				},
				{
					ID:        "2",
					Text:      "スレッド返信1-1",
					UserID:    "U2",
					ChannelID: "C1",
					Timestamp: now,
					ThreadTS:  "1", // スレッド1への返信
					IsBot:     false,
				},
				{
					ID:        "3",
					Text:      "スレッド返信1-2",
					UserID:    "U3",
					ChannelID: "C1",
					Timestamp: now,
					ThreadTS:  "1", // スレッド1への返信
					IsBot:     false,
				},
				{
					ID:        "4",
					Text:      "スレッド親2",
					UserID:    "U1",
					ChannelID: "C1",
					Timestamp: now,
					ThreadTS:  "4", // スレッドの親
					IsBot:     false,
				},
				{
					ID:        "5",
					Text:      "スレッド返信2-1",
					UserID:    "U2",
					ChannelID: "C1",
					Timestamp: now,
					ThreadTS:  "4", // スレッド2への返信
					IsBot:     false,
				},
			},
			users: map[string]*domain.User{
				"U1": {ID: "U1", Name: "User1"},
				"U2": {ID: "U2", Name: "User2"},
				"U3": {ID: "U3", Name: "User3"},
			},
			wantEmojiTop:  0,
			wantMsgTop:    0,
			wantUserTop:   3,
			wantThreadTop: 2, // スレッド1が2件、スレッド2が1件
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgRepo := &mockMessageRepository{messages: tt.messages}
			userRepo := &mockUserRepository{users: tt.users}
			analyzer := NewAnalyzer(msgRepo, userRepo)

			result, err := analyzer.AnalyzeChannel(context.Background(), "C1", nil)
			if err != nil {
				t.Fatalf("AnalyzeChannel() error = %v", err)
			}

			if len(result.EmojiStats) < tt.wantEmojiTop {
				t.Errorf("EmojiStats length = %v, want at least %v", len(result.EmojiStats), tt.wantEmojiTop)
			}
			if len(result.MessageStats) < tt.wantMsgTop {
				t.Errorf("MessageStats length = %v, want at least %v", len(result.MessageStats), tt.wantMsgTop)
			}
			if len(result.UserStats) < tt.wantUserTop {
				t.Errorf("UserStats length = %v, want at least %v", len(result.UserStats), tt.wantUserTop)
			}
			if len(result.ThreadStats) < tt.wantThreadTop {
				t.Errorf("ThreadStats length = %v, want at least %v", len(result.ThreadStats), tt.wantThreadTop)
			}
		})
	}
}
