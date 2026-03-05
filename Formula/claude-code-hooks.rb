class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  version "2.15.0"

  on_arm do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.15.0/claude-code-hooks_darwin_arm64.tar.gz"
    sha256 "a6d0b198c0a949fc300b300cfac2bd6c3e4ed0a18e25c581fdab44b3ac6ec78e"
  end
  on_intel do
    url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.15.0/claude-code-hooks_darwin_amd64.tar.gz"
    sha256 "21dbdc590a5cbfe28777a9c0f16cc7a63d0fc6bc9c954ea5994cf52ed17e2ff5"
  end

  def install
    bin.install "claude-code-hooks"
    (share/"claude-code-hooks").install "share/hooks.json"

    bash_completion.install Utils.safe_popen_read(bin/"claude-code-hooks", "completion", "bash").strip => "claude-code-hooks"
    zsh_completion.install Utils.safe_popen_read(bin/"claude-code-hooks", "completion", "zsh").strip => "_claude-code-hooks"
    fish_completion.install Utils.safe_popen_read(bin/"claude-code-hooks", "completion", "fish").strip => "claude-code-hooks.fish"
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
