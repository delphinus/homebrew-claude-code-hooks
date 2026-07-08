class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.33.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.33.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "fee1d196609d9a0878a4e8dad9bda84cb2d6c10e046802e693de6e2f9efb24ea"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.33.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "0c92e22ed965fad14d98b39547c40f7b5d0df42f96e97819cabda981b8be62c7"
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
