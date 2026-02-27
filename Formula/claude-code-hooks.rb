class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.6.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.6.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "58695f0c487bbd6e7faaaafd9c09947878c515dd2c8388f4dcef6d47207028c6"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.6.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "8a48f9d51d83c9c23011ff31bd19c05bcb3b4e1e3f00e0071705041a138adeef"
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
