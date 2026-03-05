class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.17.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.17.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "2b3fcaa78cf636fb3e946e4f839aea44bfc55b56e841512cd902aea9a7e90881"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.17.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "70f0528317895a4d2ebe6d15d2bb98d24d04ab62d057fa532dc916b30620dc90"
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
