import Foundation

/// 通知クリック / URL スキームで受け取ったペインを前面化する。kitty と WezTerm に対応する。
public enum Activator {
    /// 対応するターミナル。
    public enum Term: String, Equatable {
        case kitty
        case wezterm

        /// 未知の値や未指定は wezterm 扱い。term を載せていない古い通知が通知センターに
        /// 残っていても、クリックが壊れないようにする。
        public static func parse(_ raw: String?) -> Term {
            guard let raw = raw, let term = Term(rawValue: raw.lowercased()) else { return .wezterm }
            return term
        }

        /// socket を渡すための環境変数名。クリック起因で再起動されたインスタンスは
        /// 元の環境変数を持たないので、こちらで補ってやる必要がある。
        var sockEnvName: String {
            switch self {
            case .kitty: return "KITTY_LISTEN_ON"
            case .wezterm: return "WEZTERM_UNIX_SOCKET"
            }
        }
    }

    /// activate?pane=...&sock=...&term=... のパラメータ。
    public struct Params: Equatable {
        public let pane: String
        public let sock: String?
        public let term: Term

        public init(pane: String, sock: String?, term: Term = .wezterm) {
            self.pane = pane
            self.sock = sock
            self.term = term
        }
    }

    /// claude-code-hooks://activate?pane=N&sock=...&term=kitty を解析する。
    /// host が activate で pane が非空のときのみ Params を返す。
    public static func parse(url: URL) -> Params? {
        guard url.scheme == AppURL.scheme, url.host == "activate" else { return nil }
        let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems ?? []
        guard let pane = items.first(where: { $0.name == "pane" })?.value, !pane.isEmpty else {
            return nil
        }
        let sock = items.first(where: { $0.name == "sock" })?.value
        let term = items.first(where: { $0.name == "term" })?.value
        return Params(pane: pane, sock: (sock?.isEmpty == false) ? sock : nil, term: Term.parse(term))
    }

    /// クリック時に走らせる zsh コマンド文字列を組み立てる。
    ///
    /// pane は整数のみ許可する (シェルインジェクション防止)。socket はコマンド文字列に
    /// 直書きせず環境変数で渡すため、ここには含めない。ペインの前面化が失敗しても
    /// (`;` 区切り) ターミナル自体は前面化するよう `open -a` を続ける。
    public static func command(pane: String, term: Term = .wezterm) -> String? {
        guard let id = Int(pane) else { return nil }
        switch term {
        case .kitty:
            return "kitten @ focus-window --match id:\(id) ; open -a kitty"
        case .wezterm:
            return "wezterm cli activate-pane --pane-id \(id) ; open -a WezTerm"
        }
    }

    /// pane を前面化する。socket はターミナルに応じた環境変数で子プロセスへ渡す。
    @discardableResult
    public static func run(pane: String, sock: String?, term: Term = .wezterm) -> Process? {
        guard let cmd = command(pane: pane, term: term) else { return nil }
        let task = Process()
        task.launchPath = "/bin/zsh"
        task.arguments = ["-l", "-c", cmd]
        var env = ProcessInfo.processInfo.environment
        if let sock = sock, !sock.isEmpty {
            env[term.sockEnvName] = sock
        }
        task.environment = env
        try? task.run()
        return task
    }
}
