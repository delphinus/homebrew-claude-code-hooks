class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  url "ssh://git@github.com/delphinus/homebrew-claude-code-hooks.git", tag: "v2.3.0", revision: "1653ff39c163e31a38518dc81d757e2458d6faf8", using: :git
  head "ssh://git@github.com/delphinus/homebrew-claude-code-hooks.git", branch: "main"

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
