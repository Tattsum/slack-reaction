package slack

import "testing"

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected int
	}{
		{
			name:     "標準的な形式",
			errMsg:   "slack rate limit exceeded, retry after 10s",
			expected: 10,
		},
		{
			name:     "スペースあり",
			errMsg:   "slack rate limit exceeded, retry after  10s",
			expected: 10,
		},
		{
			name:     "長いメッセージ",
			errMsg:   "メッセージ取得エラー: slack rate limit exceeded, retry after 10s",
			expected: 10,
		},
		{
			name:     "異なる秒数",
			errMsg:   "slack rate limit exceeded, retry after 5s",
			expected: 5,
		},
		{
			name:     "retry afterがない",
			errMsg:   "slack rate limit exceeded",
			expected: 0,
		},
		{
			name:     "数値が不正",
			errMsg:   "slack rate limit exceeded, retry after abc",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRetryAfter(tt.errMsg)
			if result != tt.expected {
				t.Errorf("extractRetryAfter(%q) = %d, want %d", tt.errMsg, result, tt.expected)
			}
		})
	}
}
