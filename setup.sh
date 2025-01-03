#!/bin/bash

# エラーが発生した場合にスクリプトを終了する
set -e

# env.sh が存在しない場合は作成する
ENV_FILE="/home/isucon/env.sh"
if [ ! -e "$ENV_FILE" ]; then
  echo "Creating $ENV_FILE..."
  touch "$ENV_FILE"
  echo "$ENV_FILE を作成しました。"
else
  echo "$ENV_FILE は既に存在します。"
fi

# Makefile のコマンドでセットアップを行う
echo "Running setup commands from Makefile..."
make setup
make set-as-s1
make get-conf

# SSH 公開鍵を表示する
SSH_KEY_PATH="$HOME/.ssh/id_ed25519.pub"
if [ -e "$SSH_KEY_PATH" ]; then
  echo "--- SSH Public Key ---"
  cat "$SSH_KEY_PATH"
  echo "----------------------"
else
  echo "SSH 公開鍵が見つかりません ($SSH_KEY_PATH)。"
fi
