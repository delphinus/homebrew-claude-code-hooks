class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.5.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "b2c7507bb7c03ffa2a5858f7e294e63d5ecdd5d98024a802e36cb355bc054802"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "77c97ecee0bd42fd5d5fdc1f112064ed03c9966ba8f70e1b77451d6eb40cf270"
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
