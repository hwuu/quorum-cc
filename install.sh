#!/bin/bash
# quorum-cc 安装脚本
# 用法: curl -fsSL https://github.com/hwuu/quorum-cc/releases/latest/download/install.sh | bash
set -e

INSTALL_DIR="${HOME}/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in x86_64) ARCH="amd64" ;; aarch64|arm64) ARCH="arm64" ;; esac

RELEASE_URL="https://github.com/hwuu/quorum-cc/releases/latest/download/quorum-cc-${OS}-${ARCH}"
echo "Downloading quorum-cc for ${OS}/${ARCH}..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$RELEASE_URL" -o "$INSTALL_DIR/quorum-cc"
chmod +x "$INSTALL_DIR/quorum-cc"
echo "✅ quorum-cc installed to $INSTALL_DIR/quorum-cc"

# 检查 PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "⚠️  $INSTALL_DIR is not in PATH. Add it:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

# 设置 shell 补全
SHELL_NAME=$(basename "$SHELL")
COMPLETION_LINE='source <(quorum-cc completion '"$SHELL_NAME"')'

case "$SHELL_NAME" in
  bash)
    RC_FILE="$HOME/.bashrc"
    ;;
  zsh)
    RC_FILE="$HOME/.zshrc"
    ;;
  *)
    echo "Run 'quorum-cc init' to get started"
    exit 0
    ;;
esac

if [ -f "$RC_FILE" ] && ! grep -q 'quorum-cc completion' "$RC_FILE"; then
  echo "" >> "$RC_FILE"
  echo "# quorum-cc shell completion" >> "$RC_FILE"
  echo "$COMPLETION_LINE" >> "$RC_FILE"
  echo "✅ Shell 补全已添加到 $RC_FILE（重新打开终端或执行 source $RC_FILE 生效）"
else
  echo "ℹ️  Shell 补全已存在于 $RC_FILE"
fi

echo "Run 'quorum-cc init' to get started"
