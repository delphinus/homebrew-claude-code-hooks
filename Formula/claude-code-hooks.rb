class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.13.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.13.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "70d56d10796e8f838a1c3f121b0c446a4dd2f598211b176302ce6a7e5b009ff2"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.13.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "06ed0fdb9396828132145aaf270d383fdf67e5aa1a3c266df6a0ccd6b68e6378"
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
