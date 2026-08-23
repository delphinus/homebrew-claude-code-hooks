import ClaudeNotifyLib
import Foundation

var passed = 0
var failed = 0

func assert(_ condition: Bool, _ message: String, file: String = #file, line: Int = #line) {
    if !condition {
        fputs("FAIL: \(message) (\(file):\(line))\n", stderr)
        failed += 1
    }
}

func test(_ name: String, _ body: () -> Void) {
    body()
    passed += 1
    print("  PASS: \(name)")
}

print("Running tests...")

test("command: wezterm (既定) -> activate-pane then open") {
    let cmd = Activator.command(pane: "10")
    assert(cmd == "wezterm cli activate-pane --pane-id 10 ; open -a WezTerm", "unexpected command: \(cmd ?? "nil")")
}

test("command: kitty -> focus-window then open") {
    let cmd = Activator.command(pane: "10", term: .kitty)
    assert(cmd == "kitten @ focus-window --match id:10 ; open -a kitty", "unexpected command: \(cmd ?? "nil")")
}

test("command: non-integer pane is rejected (injection guard)") {
    for term in [Activator.Term.wezterm, .kitty] {
        assert(Activator.command(pane: "10; rm -rf /", term: term) == nil, "must reject non-integer pane")
        assert(Activator.command(pane: "", term: term) == nil, "must reject empty pane")
        assert(Activator.command(pane: "abc", term: term) == nil, "must reject alpha pane")
    }
}

test("Term.parse: 未知 / 未指定は wezterm に倒す") {
    assert(Activator.Term.parse("kitty") == .kitty, "kitty")
    assert(Activator.Term.parse("KITTY") == .kitty, "大文字も受ける")
    assert(Activator.Term.parse("wezterm") == .wezterm, "wezterm")
    assert(Activator.Term.parse(nil) == .wezterm, "未指定は wezterm")
    assert(Activator.Term.parse("ghostty") == .wezterm, "未知は wezterm")
}

test("parse: valid activate URL with pane and sock") {
    let url = URL(string: "claude-code-hooks://activate?pane=10&sock=/tmp/w.sock")!
    let p = Activator.parse(url: url)
    assert(p == Activator.Params(pane: "10", sock: "/tmp/w.sock"), "unexpected params: \(String(describing: p))")
}

test("parse: pane only (no sock) -> sock nil, term は wezterm") {
    let url = URL(string: "claude-code-hooks://activate?pane=7")!
    let p = Activator.parse(url: url)
    assert(p == Activator.Params(pane: "7", sock: nil), "unexpected params: \(String(describing: p))")
}

test("parse: term=kitty を拾う") {
    let url = URL(string: "claude-code-hooks://activate?pane=7&sock=unix:/tmp/kitty-1&term=kitty")!
    let p = Activator.parse(url: url)
    assert(
        p == Activator.Params(pane: "7", sock: "unix:/tmp/kitty-1", term: .kitty),
        "unexpected params: \(String(describing: p))"
    )
}

test("parse: wrong scheme -> nil") {
    let url = URL(string: "https://activate?pane=10")!
    assert(Activator.parse(url: url) == nil, "must reject non-matching scheme")
}

test("parse: wrong host -> nil") {
    let url = URL(string: "claude-code-hooks://open?pane=10")!
    assert(Activator.parse(url: url) == nil, "must reject non-activate host")
}

test("parse: missing pane -> nil") {
    let url = URL(string: "claude-code-hooks://activate?sock=/tmp/w.sock")!
    assert(Activator.parse(url: url) == nil, "must reject when pane missing")
}

print("\n\(passed) passed, \(failed) failed")

if failed > 0 {
    exit(1)
}
