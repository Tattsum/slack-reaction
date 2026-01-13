.PHONY: help lint lint-fix test build clean

# デフォルトターゲット
.DEFAULT_GOAL := help

# 変数定義
MARKDOWN_FILES := $(shell find . -name "*.md" -not -path "./node_modules/*" -not -path "./.git/*")

help: ## 利用可能なタスクの一覧を表示
	@echo "利用可能なタスク:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

lint: ## Markdownファイルのlinterチェックを実行
	@echo "Markdown linterチェックを実行中..."
	@npx --yes markdownlint-cli2 "**/*.md" "#node_modules"
	@echo "✅ Linterチェック完了"

lint-fix: ## Markdownファイルのlinterエラーを自動修正（可能な場合）
	@echo "Markdown linterの自動修正を実行中..."
	@npx --yes markdownlint-cli2 --fix "**/*.md" "#node_modules" || true
	@echo "✅ 自動修正完了"

test: ## テストを実行
	@echo "テストを実行中..."
	@go test -v ./...

build: ## アプリケーションをビルド
	@echo "アプリケーションをビルド中..."
	@go build -o slack-reaction main.go
	@echo "✅ ビルド完了: ./slack-reaction"

clean: ## ビルド成果物を削除
	@echo "ビルド成果物を削除中..."
	@rm -f slack-reaction
	@echo "✅ クリーンアップ完了"

check: lint ## すべてのチェックを実行（lintを含む）
	@echo "✅ すべてのチェック完了"

ci: lint test ## CI/CD用: lintとtestを実行
	@echo "✅ CIチェック完了"
