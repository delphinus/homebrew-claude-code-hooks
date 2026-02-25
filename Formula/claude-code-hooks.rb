class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  url "https://github.com/delphinus/homebrew-claude-code-hooks/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
  head "https://github.com/delphinus/homebrew-claude-code-hooks.git", branch: "main"

  depends_on "jq"

  def install
    bin.install "bin/claude-obsidian-save"
    bin.install "bin/claude-obsidian-backfill-links"
    bin.install "bin/claude-setup-hooks"
    bin.install "bin/claude-notify"
    (share/"claude-code-hooks").install "share/hooks.json"
  end

  def caveats
    <<~EOS
      hooks.json をインストールしました。以下のコマンドで Claude Code に適用してください:

        claude-setup-hooks

      差分を事前に確認するには:

        claude-setup-hooks --diff
    EOS
  end
end
