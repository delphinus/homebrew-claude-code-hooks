class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.14.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.14.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "b4d2429479e9664c586984d4e6f6e2a48a70c6427558811e588fb7be2d206d0e"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.14.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "98598fcf778b06474b9d92c95bfdcf18a4bc00237d5f5d18ce7fb87a2faed594"
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
