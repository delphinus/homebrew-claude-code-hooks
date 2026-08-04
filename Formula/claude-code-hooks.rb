class ClaudeCodeHooks < Formula
  desc "Claude Code hooks for Obsidian integration and notifications"
  homepage "https://github.com/delphinus/homebrew-claude-code-hooks"
  url "https://github.com/delphinus/homebrew-claude-code-hooks/releases/download/v2.36.1/claude-code-hooks.tar.gz"
  sha256 "90be943614db1a13e08ca25ae449a5ce84fb6d0fa9ac0d166386ebc55678346a"
  version "2.36.1"

  depends_on :macos

  def install
    bin.install "claude-code-hooks"
    prefix.install "claude-code-hooks-notify.app"
    bin.write_exec_script prefix/"claude-code-hooks-notify.app/Contents/MacOS/claude-code-hooks-notify"
    (share/"claude-code-hooks").install "share/hooks.json"

    generate_completions_from_executable(bin/"claude-code-hooks", "completion", shells: [:bash, :zsh, :fish])
  end

  def post_install
    system "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister",
 "-R", prefix/"claude-code-hooks-notify.app"
  end

  def caveats
    <<~EOS
      インストール後、以下のコマンドで Claude Code に hooks を適用してください:

        claude-code-hooks setup

      差分を事前に確認するには:

        claude-code-hooks setup --diff

      シェル補完は自動的にインストールされています（Bash / Zsh / Fish）。

      通知をクリックすると、その Claude Code が動いている WezTerm のペインが前面化します。
      初回の通知時に通知の許可を求められるので許可してください
      （システム設定 > 通知 > claude-code-hooks-notify）。

      [推奨] Obsidian の Advanced URI プラグインを導入すると、
      ノートが新しいタブで開くようになります:
        https://github.com/Vinzent03/obsidian-advanced-uri
    EOS
  end
end
