require_relative "../lib/private_strategy"

class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.5.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_arm64.tar.gz",
        using: GitHubPrivateRepositoryReleaseDownloadStrategy
    sha256 "f03c436bdc0176c54ce8b7e5af8dd0628ac3a9fcb5c5528a6f258e7a84047586"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.5.0/claude-code-hooks_darwin_amd64.tar.gz",
        using: GitHubPrivateRepositoryReleaseDownloadStrategy
    sha256 "411766210496a56bc58652b0ee9bf4d8409289637a3bdb030cbcc1eba4918e30"
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
