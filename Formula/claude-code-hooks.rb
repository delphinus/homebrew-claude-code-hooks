class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.8.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.8.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "ee76b005f2e80417d41df97a885ebf97ac415136b97b6cc4d1a718febc33e25a"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.8.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "56c7347cd00de5c0d9b4e3681c34c71c314873c4d27190e1fa588e99cb9eadfc"
  end

  def install
    bin.install "claude-code-hooks"
    (share/"claude-code-hooks").install "share/hooks.json"
  end

  def caveats
    <<~EOS
      インストール後、以下のコマンドで Claude Code に hooks を適用してください:

        claude-code-hooks setup

      差分を事前に確認するには:

        claude-code-hooks setup --diff
    EOS
  end
end
