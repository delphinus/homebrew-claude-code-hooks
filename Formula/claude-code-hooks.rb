class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  url "git@github.com:delphinus/homebrew-claude-code-hooks.git", tag: "v2.0.0", revision: "c1242dfe34f9c3434cf30b15b37e61a7f817eade", using: :git
  head "git@github.com:delphinus/homebrew-claude-code-hooks.git", branch: "main", using: :git

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/claude-code-hooks"
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
