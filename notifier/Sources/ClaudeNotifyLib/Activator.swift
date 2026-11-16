import Foundation

/// 通知クリック / URL スキームで受け取った kitty のウィンドウを前面化する。
public enum Activator {
    /// socket を渡すための環境変数名。クリック起因で再起動されたインスタンスは
    /// 元の環境変数を持たないので、こちらで補ってやる必要がある。
    static let sockEnvName = "KITTY_LISTEN_ON"

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
    /// 以前の通知が載せていた term パラメータは読まない。
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
    /// 直書きせず環境変数で渡すため、ここには含めない。ウィンドウの前面化が失敗しても
    /// (`;` 区切り) kitty 自体は前面化するよう `open -a` を続ける。
    public static func command(pane: String) -> String? {
        guard let id = Int(pane) else { return nil }
        return "kitten @ focus-window --match id:\(id) ; open -a kitty"
    }

    /// pane を前面化する。socket は KITTY_LISTEN_ON で子プロセスへ渡す。
    @discardableResult
    public static func run(pane: String, sock: String?) -> Process? {
        guard let cmd = command(pane: pane) else { return nil }
        let task = Process()
        task.launchPath = "/bin/zsh"
        task.arguments = ["-l", "-c", cmd]
        var env = ProcessInfo.processInfo.environment
        if let sock = sock, !sock.isEmpty {
            env[sockEnvName] = sock
        }
        task.environment = env
        try? task.run()
        return task
    }
}
