class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.7.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.7.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "1fc761184179d0ef74a4cfb3f012c3331169e84f6b938b01b8e799eec0faeef1"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.7.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "6b6d4d29607dffb08b3d9aede7e71c541371ecc49a3f1df0cc96b525fef86da7"
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
