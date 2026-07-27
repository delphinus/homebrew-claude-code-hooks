import ClaudeNotifyLib
import Foundation

let notificationSystem = NotificationSystem()

let args = CommandLine.arguments

func value(for flag: String) -> String? {
    guard let i = args.firstIndex(of: flag), i + 1 < args.count else { return nil }
    return args[i + 1]
}

// サブコマンド無しでの起動 = 通知クリック / URL スキーム起因の再起動。
// ここで didReceive / open(URL) を短いイベントループで拾って即終了する。
// (post 等の明示コマンド時はこの待機を挟まず、通知投稿だけ行って素早く抜ける)
let hasCommand = args.contains("post") || args.contains("--test")
    || args.contains("--help") || args.contains("-h")
if !hasCommand {
    _ = notificationSystem.handlePendingActions()
    exit(0)
}

if args.contains("--help") || args.contains("-h") {
    print("claude-code-hooks-notify: クリックで WezTerm のペインを前面化する macOS 通知ヘルパー")
    print()
    print("Usage:")
    print("  claude-code-hooks-notify post --title T --message M [--subtitle S] [--pane N]")
    print("      通知を投稿する。--pane を付けるとクリック時にその WezTerm ペインを前面化する。")
    print("      --subtitle には通知元のタブ (\"<タブ番号>: <タブタイトル>\") 等を載せる。")
    print("      socket は環境変数 WEZTERM_UNIX_SOCKET から読む。")
    print("  claude-code-hooks-notify --test")
    print("      テスト通知を投稿する (--pane に WEZTERM_PANE を使う)。")
    print()
    print("URL scheme (通知クリックの代替):")
    print("  open \"claude-code-hooks://activate?pane=N&sock=/path/to/sock\"")
    exit(0)
}

let pane = value(for: "--pane") ?? ProcessInfo.processInfo.environment["WEZTERM_PANE"]
let sock = ProcessInfo.processInfo.environment["WEZTERM_UNIX_SOCKET"]

if args.contains("--test") {
    notificationSystem.post(
        title: "Claude Code",
        subtitle: value(for: "--subtitle") ?? pane.map { "ペイン \($0)" },
        message: "テスト通知: クリックで WezTerm を前面化します",
        pane: pane,
        sock: sock
    )
} else if args.contains("post") {
    let title = value(for: "--title") ?? "Claude Code"
    let message = value(for: "--message") ?? "通知"
    notificationSystem.post(
        title: title,
        subtitle: value(for: "--subtitle"),
        message: message,
        pane: pane,
        sock: sock
    )
} else {
    FileHandle.standardError.write(Data("usage: claude-code-hooks-notify post --title T --message M [--subtitle S] [--pane N]\n".utf8))
    exit(1)
}
