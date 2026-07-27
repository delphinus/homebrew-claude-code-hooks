import Foundation

/// 通知クリック / URL スキームで受け取ったペインを WezTerm で前面化する。
public enum Activator {
    /// activate?pane=...&sock=... のパラメータ。
    public struct Params: Equatable {
        public let pane: String
        public let sock: String?

        public init(pane: String, sock: String?) {
            self.pane = pane
            self.sock = sock
        }
    }

    /// claude-code-hooks://activate?pane=N&sock=... を解析する。
    /// host が activate で pane が非空のときのみ Params を返す。
    public static func parse(url: URL) -> Params? {
        guard url.scheme == AppURL.scheme, url.host == "activate" else { return nil }
        let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems ?? []
        guard let pane = items.first(where: { $0.name == "pane" })?.value, !pane.isEmpty else {
            return nil
        }
        let sock = items.first(where: { $0.name == "sock" })?.value
        return Params(pane: pane, sock: (sock?.isEmpty == false) ? sock : nil)
    }

    /// クリック時に走らせる zsh コマンド文字列を組み立てる。
    ///
    /// pane は整数のみ許可する (シェルインジェクション防止)。socket はコマンド文字列に
    /// 直書きせず環境変数で渡すため、ここには含めない。activate-pane が失敗しても
    /// (`;` 区切り) WezTerm 自体は前面化するよう `open -a WezTerm` を続ける。
    public static func command(pane: String) -> String? {
        guard let id = Int(pane) else { return nil }
        return "wezterm cli activate-pane --pane-id \(id) ; open -a WezTerm"
    }

    /// pane を前面化する。socket は WEZTERM_UNIX_SOCKET として子プロセスの環境に渡す
    /// (クリック起因の再起動インスタンスは元の環境変数を持たないため)。
    @discardableResult
    public static func run(pane: String, sock: String?) -> Process? {
        guard let cmd = command(pane: pane) else { return nil }
        let task = Process()
        task.launchPath = "/bin/zsh"
        task.arguments = ["-l", "-c", cmd]
        var env = ProcessInfo.processInfo.environment
        if let sock = sock, !sock.isEmpty {
            env["WEZTERM_UNIX_SOCKET"] = sock
        }
        task.environment = env
        try? task.run()
        return task
    }
}
