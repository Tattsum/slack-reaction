# コントリビューションガイドライン

このドキュメントでは、`slack-reaction` プロジェクトへの貢献方法を説明します。

## 開発環境のセットアップ

### 前提条件

1. Go 1.24.2以上がインストールされていること
2. Node.jsとnpmがインストールされていること（Markdown linter用）
3. Slack User Tokenが設定されていること（詳細は[Slackトークン取得方法](slack-token-setup.md)を参照）

### セットアップ手順

1. リポジトリをクローン:

   ```bash
   git clone https://github.com/Tattsum/slack-reaction.git
   cd slack-reaction
   ```

2. 依存関係のインストール:

   ```bash
   go mod tidy
   ```

3. 環境変数の設定:

   ```bash
   export SLACK_USER_TOKEN="your-slack-user-token"
   ```

## 開発ワークフロー

### 1. ブランチの作成

新しい機能やバグ修正を行う場合は、新しいブランチを作成してください:

```bash
git checkout -b feature/your-feature-name
# または
git checkout -b fix/your-bug-fix-name
```

### 2. コードの変更

- ドメイン駆動設計を前提に設計してください
- テスト駆動開発（TDD）を心がけてください
- セキュリティとパフォーマンスを最大限考慮してください

### 3. コミット前のチェック

**重要**: コードをコミットする前に、必ず以下のチェックを実行してください。

#### Linterチェック

```bash
make lint
```

または

```bash
make check
```

このコマンドは以下のチェックを実行します:

- Markdownファイルのlinterチェック（markdownlint）

エラーが検出された場合は、修正してからコミットしてください。

#### 自動修正

一部のlinterエラーは自動修正できます:

```bash
make lint-fix
```

自動修正後、再度 `make lint` を実行して、すべてのエラーが解消されたことを確認してください。

#### テストの実行

```bash
make test
```

または

```bash
go test -v ./...
```

### 4. コミット

コミットメッセージは明確で説明的にしてください:

```bash
git add .
git commit -m "feat: 新機能の説明"
# または
git commit -m "fix: バグ修正の説明"
# または
git commit -m "docs: ドキュメントの更新"
```

### 5. プッシュとプルリクエスト

```bash
git push origin feature/your-feature-name
```

その後、GitHubでプルリクエストを作成してください。

## コーディング規約

### Go コーディング規約

- `gofmt` でフォーマットされたコードを使用してください
- `golint` や `golangci-lint` の推奨事項に従ってください
- エラーハンドリングを適切に行ってください
- コメントは日本語で記述してください（公開APIは英語も検討）

### Markdown コーディング規約

- markdownlintのルールに準拠してください
- コードブロックには言語指定を必ず含めてください
- リストの前後には空行を入れてください
- ファイルの末尾には改行を入れてください

## テスト

### テストの書き方

- テスト駆動開発（TDD）を推奨します
- テストファイルは `*_test.go` という命名規則に従ってください
- テスト関数は `TestXxx` という形式で命名してください

### テストの実行方法

```bash
make test
```

## ビルド

### ローカルビルド

```bash
make build
```

実行ファイル `slack-reaction` が生成されます。

### クリーンアップ

```bash
make clean
```

ビルド成果物を削除します。

## 利用可能なMakeタスク

プロジェクトには以下のMakeタスクが用意されています:

- `make help` - 利用可能なタスクの一覧を表示
- `make lint` - Markdownファイルのlinterチェックを実行
- `make lint-fix` - Markdownファイルのlinterエラーを自動修正
- `make test` - テストを実行
- `make build` - アプリケーションをビルド
- `make clean` - ビルド成果物を削除
- `make check` - すべてのチェックを実行（lintを含む）
- `make ci` - CI/CD用: lintとtestを実行

## プルリクエストのチェックリスト

プルリクエストを作成する前に、以下を確認してください:

- [ ] コードが `make lint` をパスしている
- [ ] すべてのテストが `make test` でパスしている
- [ ] 新しい機能には適切なテストが追加されている
- [ ] ドキュメントが更新されている（必要に応じて）
- [ ] コミットメッセージが明確で説明的である
- [ ] セキュリティ上の懸念がないか確認している

## CI/CD

プルリクエストが作成されると、自動的に以下のチェックが実行されます:

- Markdown linterチェック
- Go テストの実行

すべてのチェックがパスする必要があります。

## 質問や問題

質問や問題がある場合は、GitHubのIssuesで報告してください。

## ライセンス

このプロジェクトへの貢献は、プロジェクトのライセンス（MIT License）の下で公開されることに同意したものとみなされます。
