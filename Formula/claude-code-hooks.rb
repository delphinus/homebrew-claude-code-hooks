class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.10.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.10.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "0851c690999d7fd22d24f8d9accbf466fa03aa58cc2ca37b1fe6e6b7efcec963"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.10.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "0b3975aef5d84b573e729aafdd320d4abfea26913a887fdcfab5ffeb5d42c00d"
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
