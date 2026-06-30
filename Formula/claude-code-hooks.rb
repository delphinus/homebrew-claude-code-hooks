class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.30.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.30.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "4c16dfe54110453d4a50f561ca37b2594ec0f266ead22b268f38757c6933c6f6"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.30.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "7c53ef59fe2a7a7e9053b256948f161cb87aedfc85ea08c8bc818e01e90bb7a4"
  end

  def install
    bin.install "claude-code-hooks"
    (share/"claude-code-hooks").install "share/hooks.json"

    generate_completions_from_executable(bin/"claude-code-hooks", "completion", shells: [:bash, :zsh, :fish])
  end

  def caveats
    <<~EOS
      インストール後、以下のコマンドで Claude Code に hooks を適用してください:

        claude-code-hooks setup

      差分を事前に確認するには:

        claude-code-hooks setup --diff

      シェル補完は自動的にインストールされています（Bash / Zsh / Fish）。

      [推奨] Obsidian の Advanced URI プラグインを導入すると、
      ノートが新しいタブで開くようになります:
        https://github.com/Vinzent03/obsidian-advanced-uri
    EOS
  end
end
