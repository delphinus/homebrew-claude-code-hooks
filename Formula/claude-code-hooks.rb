class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.32.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.32.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "4428010e6afd8e29f34fd8545be1284d220201f3c5fcc47afe39c2659854fef4"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.32.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "81ac97aa38ad069aca6ff266566fa07798f19ea6db884d1373528e3cad554e49"
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
