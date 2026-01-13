# APIコール数削減の検討

## 現状の問題

現在のユーザー分析機能では、全チャンネル（2514チャンネル）からメッセージを取得してから、ユーザーIDでフィルタリングしています。これにより、以下の問題が発生しています：

- **大量のAPIコール**: 各チャンネルから全メッセージを取得するため、APIコール数が非常に多い
- **レート制限**: 頻繁にレート制限エラーが発生し、処理時間が長くなる
- **非効率**: ユーザーが投稿していないチャンネルからも全メッセージを取得している

## 改善案

### 1. Slack Search APIの活用

Slack Search API (`search.messages`) を使用することで、ユーザーIDでフィルタリングしたメッセージを直接取得できます。

**メリット**:
- APIコール数を大幅に削減（全チャンネル横断で1回のAPIコールで検索可能）
- レート制限の影響を最小化
- 処理時間の短縮

**デメリット**:
- Search APIは有料プランでのみ利用可能な場合がある
- 検索結果の制限（最大100件/ページ）がある可能性
- スレッドの返信が含まれない可能性

**実装例**:
```go
params := slack.NewSearchParameters()
params.Query = fmt.Sprintf("from:%s", userID)
params.Count = 100
params.Page = 1

searchResults, err := api.SearchMessages(params)
```

### 2. slack-goライブラリのアップデート

現在のバージョン: `v0.16.0`  
最新バージョン: `v0.17.3`

**アップデートのメリット**:
- バグ修正
- パフォーマンス改善
- 新機能の利用

**注意点**:
- 破壊的変更の有無を確認
- テストの実行が必要

### 3. 並行処理の最適化

現在は順次処理していますが、並行処理を導入することで処理時間を短縮できます。

**注意点**:
- レート制限を考慮した並行数の制限が必要
- セマフォを使用して同時実行数を制御

## 推奨される実装順序

1. **slack-goライブラリのアップデート** (低リスク)
   - バージョンアップ
   - テスト実行
   - 破壊的変更の確認

2. **Search APIの検証** (中リスク)
   - Search APIが利用可能か確認
   - 検索結果の制限を確認
   - スレッド返信の取得方法を確認

3. **Search APIの実装** (中リスク)
   - `FindByUser`メソッドをSearch APIベースに変更
   - フォールバック機能の実装（Search APIが使えない場合）

4. **並行処理の最適化** (低リスク)
   - 並行処理の導入
   - レート制限対応の強化

## 参考リンク

- [Slack Search API Documentation](https://api.slack.com/methods/search.messages)
- [slack-go GitHub Repository](https://github.com/slack-go/slack)
- [Slack API Rate Limits](https://api.slack.com/docs/rate-limits)
