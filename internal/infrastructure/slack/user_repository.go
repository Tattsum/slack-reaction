package slack

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Tattsum/slack-reaction/internal/domain"
	"github.com/slack-go/slack"
)

// UserRepository はSlack APIを使用してユーザー情報を取得するリポジトリ
type UserRepository struct {
	client *slack.Client
}

// NewUserRepository は新しいUserRepositoryを作成する
func NewUserRepository(client *slack.Client) *UserRepository {
	return &UserRepository{
		client: client,
	}
}

// FindAll はすべてのユーザーを取得する
func (r *UserRepository) FindAll(ctx context.Context) (map[string]*domain.User, error) {
	users, err := r.client.GetUsers()
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]*domain.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &domain.User{
			ID:          users[i].ID,
			Name:        users[i].Name,
			DisplayName: users[i].Profile.DisplayName,
			RealName:    users[i].RealName,
		}
	}

	return userMap, nil
}

// FindByIDs は指定されたIDのユーザーを取得する
func (r *UserRepository) FindByIDs(ctx context.Context, userIDs []string) (map[string]*domain.User, error) {
	if len(userIDs) == 0 {
		return make(map[string]*domain.User), nil
	}

	// まず全ユーザーリストを取得を試みる
	allUsers, err := r.FindAll(ctx)
	if err == nil {
		// 成功した場合は必要なユーザーのみを抽出
		result := make(map[string]*domain.User, len(userIDs))
		for _, userID := range userIDs {
			if user, exists := allUsers[userID]; exists {
				result[userID] = user
			} else {
				// 見つからない場合はIDのみで作成
				result[userID] = &domain.User{
					ID:   userID,
					Name: userID,
				}
			}
		}
		return result, nil
	}

	// エラーの場合は個別取得にフォールバック
	return r.fetchUsersIndividual(ctx, userIDs)
}

// fetchUsersIndividual は個別にユーザー情報を取得する（フォールバック用）
func (r *UserRepository) fetchUsersIndividual(ctx context.Context, userIDs []string) (map[string]*domain.User, error) {
	userMap := make(map[string]*domain.User, len(userIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, 10) // 最大10並行

	for _, userID := range userIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			userInfo, err := r.client.GetUserInfo(id)
			mu.Lock()
			defer mu.Unlock()

			if err == nil && userInfo != nil {
				userMap[id] = &domain.User{
					ID:          userInfo.ID,
					Name:        userInfo.Name,
					DisplayName: userInfo.Profile.DisplayName,
					RealName:    userInfo.RealName,
				}
			} else {
				// エラーの場合はIDのみで作成
				userMap[id] = &domain.User{
					ID:   id,
					Name: id,
				}
			}
		}(userID)
	}

	wg.Wait()
	return userMap, nil
}

// FindByName はユーザー名からユーザーを検索する
func (r *UserRepository) FindByName(ctx context.Context, name string) (*domain.User, error) {
	allUsers, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// 名前、表示名、実名で検索（大文字小文字を区別しない）
	nameLower := strings.ToLower(name)
	for _, user := range allUsers {
		if strings.ToLower(user.Name) == nameLower ||
			strings.ToLower(user.DisplayName) == nameLower ||
			strings.ToLower(user.RealName) == nameLower {
			return user, nil
		}
	}

	return nil, fmt.Errorf("ユーザー '%s' が見つかりません", name)
}
