package domain

// User はSlackユーザーを表すドメインモデル
type User struct {
	ID        string
	Name      string
	DisplayName string
	RealName  string
}

// UserStats はユーザーの統計情報を表すドメインモデル
type UserStats struct {
	UserID   string
	UserName string
	Count    int
}

// GetDisplayName は表示名を優先順位に従って返す
// 優先順位: DisplayName > RealName > Name > ID
func (u *User) GetDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.RealName != "" {
		return u.RealName
	}
	if u.Name != "" {
		return u.Name
	}
	return u.ID
}
