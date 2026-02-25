class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
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
