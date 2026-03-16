class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.24.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.24.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "410dc59bd09b10d11a5e267cab8723645cfbe5d9cee146d24c8e6334273989ca"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.24.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "68ea4877eb291bb0b1f3702491bd5a553c67fe0bcf3ba0c226c9b6bedcea0146"
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
