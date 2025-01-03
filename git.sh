#!/bin/bash

# 引数として Git リポジトリの URL を受け取る
if [ -z "$1" ]; then
  echo "Usage: $0 <repository-url>"
  exit 1
fi

REPO_URL="$1"

# git の初期化
git init

# リモートリポジトリを追加
echo "Adding remote repository: $REPO_URL"
git remote add origin "$REPO_URL"

# .gitignore ファイルを作成
echo "Creating .gitignore..."
cat << EOF >> .gitignore
php
java
node
nodejs
python
ruby
perl
rust
/tool-config/alp/notify-slack.toml
/tool-config/slow-query/notify-slack.toml
EOF

# Makefile のクエリ抽出コマンドを実行
echo "Running 'make extract-queries'..."
make extract-queries

# 初期コミット
echo "Adding and committing initial files..."
git add .
git commit -m 'first commit'

# ブランチの作成とプッシュ
echo "Setting up main branch..."
git branch -M main
git push -u origin main

echo "Git repository setup complete."
