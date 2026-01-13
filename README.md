# slack-reaction

Slackチャンネルのリアクション分析ツール

## 概要

このツールは、指定されたSlackチャンネルのメッセージとリアクションを分析し、以下の統計情報を表示します：

- 最も使用されたスタンプ TOP3
- 最もリアクションがついたメッセージ TOP3
- 最も投稿数が多いユーザー TOP10

## 使用方法

### 前提条件

1. Go 1.24.2以上がインストールされていること
2. Slack User Tokenが必要（詳細は[Slackトークン取得方法](docs/slack-token-setup.md)を参照）

### セットアップ

1. 依存関係のインストール：

   ```bash
   go mod tidy
   ```

1. Slack User Tokenの設定：

   - トークンの取得方法については、[Slackトークン取得方法](docs/slack-token-setup.md)を参照してください
   - 取得したトークンを環境変数に設定：

   ```bash
   export SLACK_USER_TOKEN="your-slack-user-token"
   ```

### 実行

```bash
# 新しいアーキテクチャ（推奨）
go run ./cmd/slack-reaction -channel <チャンネル名> [-start YYYY-MM-DD] [-end YYYY-MM-DD]

# またはビルドして実行
make build
./slack-reaction -channel <チャンネル名> [-start YYYY-MM-DD] [-end YYYY-MM-DD]
```

#### パラメータ

- `-channel`: 分析対象のSlackチャンネル名（必須）
- `-start`: 開始日時（YYYY-MM-DD形式、省略可）
- `-end`: 終了日時（YYYY-MM-DD形式、省略可）

#### 実行例

```bash
# 全期間の分析
go run ./cmd/slack-reaction -channel general

# 期間を指定した分析
go run ./cmd/slack-reaction -channel general -start 2023-01-01 -end 2023-12-31
```

## 出力例

```text
===== 最も使用されたスタンプ TOP3 =====
1位: :+1: - 45回
2位: :eyes: - 32回
3位: :smile: - 28回

===== 最もリアクションがついたメッセージ TOP3 =====
1位: 新機能のリリースについて...
リアクション数: 15

2位: チームミーティングの件で...
リアクション数: 12

3位: 今日のランチ...
リアクション数: 8

===== 最も投稿数が多いユーザー TOP10 =====
1位: 田中太郎 - 156投稿
2位: 佐藤花子 - 142投稿
3位: 鈴木一郎 - 98投稿
```

## 機能

- Slackチャンネルのメッセージとリアクションの包括的な分析
- スレッドメッセージの分析もサポート
- 期間指定による分析
- レート制限対応による安定した実行
- 並行処理によるパフォーマンス最適化

## アーキテクチャ

このプロジェクトは**ドメイン駆動設計（DDD）**と**テスト駆動開発（TDD）**の原則に基づいて設計されています。

### ディレクトリ構造

```text
slack-reaction/
├── cmd/
│   └── slack-reaction/    # アプリケーションエントリーポイント
├── internal/
│   ├── domain/            # ドメインモデル（エンティティ、値オブジェクト）
│   ├── service/           # ビジネスロジック（ユースケース）
│   └── infrastructure/    # インフラ層（Slack APIクライアント）
│       └── slack/
└── docs/                  # ドキュメント
```

### レイヤー分離

- **Domain層**: ビジネスロジックの中核となるドメインモデル
- **Service層**: ドメインロジックを組み合わせたユースケース実装
- **Infrastructure層**: 外部API（Slack API）との通信

### テスト

- **ユニットテスト**: ドメインロジックとサービスロジックのテスト
- **テストカバレッジ**: `make test-coverage` でカバレッジレポートを生成

## セキュリティ

このツールは以下のセキュリティ対策を実装しています：

- Slack APIトークンは環境変数で管理
- 機密情報がログに出力されないよう配慮
- レート制限に適切に対応

## 依存関係

- [slack-go/slack](https://github.com/slack-go/slack) v0.16.0

## ドキュメント

- [Slackトークン取得方法](docs/slack-token-setup.md) - Slack User Tokenの取得と設定方法の詳細ガイド
- [トラブルシューティング](docs/TROUBLESHOOTING.md) - よくある問題とその解決方法
- [コントリビューションガイドライン](docs/CONTRIBUTING.md) - プロジェクトへの貢献方法と開発ガイドライン

## AIエージェント向け設定

このプロジェクトには、以下のAIエージェント向けの設定ファイルが含まれています：

- **Cursor**: `.cursorrules` - Cursor AIが自動的に参照するルールファイル
- **GitHub Copilot**: `.github/COPILOT_INSTRUCTIONS.md` - GitHub Copilot向けの指示
- **Claude Code**: `.claude_instructions.md` - Claude Code向けの指示
- **Gemini**: `.gemini_instructions.md` - Gemini Code向けの指示

これらの設定により、AIエージェントは以下のルールを自動的に遵守します：

- コード変更後は必ず `make lint` を実行
- コミット前に `make check` を実行
- Markdownファイルはmarkdownlintのルールに準拠

詳細は各設定ファイルを参照してください。

## ライセンス

MIT License
