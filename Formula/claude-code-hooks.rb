class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.11.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.11.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "0f4b68492355fd6a1c4a124e78d5f86d261902932e14acafe808fbe8c89bf29d"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.11.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "27ea61959611c0951d5ffe635645454cdc93b81f0bf3eccfcbcb56695d383b25"
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

      [推奨] Obsidian の Advanced URI プラグインを導入すると、
      ノートが新しいタブで開くようになります:
        https://github.com/Vinzent03/obsidian-advanced-uri
    EOS
  end
end
