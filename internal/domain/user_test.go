package domain

import "testing"

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		expected string
	}{
		{
			name: "DisplayNameが優先",
			user: &User{
				ID:          "U123",
				DisplayName: "表示名",
				RealName:    "実名",
				Name:        "ユーザー名",
			},
			expected: "表示名",
		},
		{
			name: "DisplayNameが空の場合はRealName",
			user: &User{
				ID:          "U123",
				DisplayName: "",
				RealName:    "実名",
				Name:        "ユーザー名",
			},
			expected: "実名",
		},
		{
			name: "DisplayNameとRealNameが空の場合はName",
			user: &User{
				ID:          "U123",
				DisplayName: "",
				RealName:    "",
				Name:        "ユーザー名",
			},
			expected: "ユーザー名",
		},
		{
			name: "すべて空の場合はID",
			user: &User{
				ID:          "U123",
				DisplayName: "",
				RealName:    "",
				Name:        "",
			},
			expected: "U123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.GetDisplayName(); got != tt.expected {
				t.Errorf("GetDisplayName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
