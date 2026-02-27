class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.5.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "aa7720f33a4f1c7217bf45a7b53f2e5bf8e6ab478456e94eda12343ffcf83182"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "132f73fac8d92efae1d1a6cebc274d8ae65ede45f03c8964c2ef9b671c60a967"
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
