package domain

import (
	"testing"
	"time"
)

func TestMessage_HasReactions(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		expected bool
	}{
		{
			name: "リアクションあり",
			message: &Message{
				Reactions: []Reaction{
					{Name: "thumbsup", Count: 5},
				},
			},
			expected: true,
		},
		{
			name: "リアクションなし",
			message: &Message{
				Reactions: []Reaction{},
			},
			expected: false,
		},
		{
			name: "リアクションがnil",
			message: &Message{
				Reactions: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.HasReactions(); got != tt.expected {
				t.Errorf("HasReactions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessage_TotalReactionCount(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		expected int
	}{
		{
			name: "複数のリアクション",
			message: &Message{
				Reactions: []Reaction{
					{Name: "thumbsup", Count: 5},
					{Name: "smile", Count: 3},
					{Name: "heart", Count: 2},
				},
			},
			expected: 10,
		},
		{
			name: "リアクションなし",
			message: &Message{
				Reactions: []Reaction{},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.TotalReactionCount(); got != tt.expected {
				t.Errorf("TotalReactionCount() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessage_IsThreadReply(t *testing.T) {
	now := time.Now()
	timestamp := now.Format("1504840306.000009")
	threadTS := "1234567890.123456"

	tests := []struct {
		name     string
		message  *Message
		expected bool
	}{
		{
			name: "スレッドの返信",
			message: &Message{
				ID:       timestamp,
				ThreadTS: threadTS,
			},
			expected: true,
		},
		{
			name: "通常のメッセージ",
			message: &Message{
				ID:       timestamp,
				ThreadTS: "",
			},
			expected: false,
		},
		{
			name: "スレッドの親メッセージ",
			message: &Message{
				ID:       timestamp,
				ThreadTS: timestamp,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsThreadReply(); got != tt.expected {
				t.Errorf("IsThreadReply() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessage_IsThreadParent(t *testing.T) {
	now := time.Now()
	timestamp := now.Format("1504840306.000009")

	tests := []struct {
		name     string
		message  *Message
		expected bool
	}{
		{
			name: "スレッドの親メッセージ",
			message: &Message{
				ID:       timestamp,
				ThreadTS: timestamp,
			},
			expected: true,
		},
		{
			name: "通常のメッセージ",
			message: &Message{
				ID:       timestamp,
				ThreadTS: "",
			},
			expected: false,
		},
		{
			name: "スレッドの返信",
			message: &Message{
				ID:       timestamp,
				ThreadTS: "1234567890.123456",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsThreadParent(); got != tt.expected {
				t.Errorf("IsThreadParent() = %v, want %v", got, tt.expected)
			}
		})
	}
}
