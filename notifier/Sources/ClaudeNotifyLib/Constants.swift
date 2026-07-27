import Foundation

enum AppURL {
    /// 通知クリック相当のアクションを外部からも叩けるようにする URL スキーム。
    /// 例: open "claude-code-hooks://activate?pane=10&sock=/path/to/sock"
    static let scheme = "claude-code-hooks"
    static let repo: URL? = .init(string: "https://github.com/delphinus/homebrew-claude-code-hooks")
}
