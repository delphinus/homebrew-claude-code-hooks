import Foundation
import AppKit
import UserNotifications

public final class NotificationSystem {
    private var handler: ActionHandler?
    private let center = UNUserNotificationCenter.current()

    public init() {}

    /// クリック起因でバンドルが再起動されたときに、配送済みレスポンス (didReceive) を
    /// 短いイベントループで拾って処理する。処理したら true。
    public func handlePendingActions() -> Bool {
        let handler = ensureSetup()
        runEventLoop(handler: handler, timeoutSeconds: 5.0)
        waitForActivation(handler: handler)
        return handler.actionHandled
    }

    /// クリック可能な通知を投稿する。pane / sock は userInfo に載せ、クリック時に読む。
    /// subtitle には通知元のタブ ("<タブ番号>: <タブタイトル>") を載せる。
    /// 配送完了まで待って return する (クリック待ちの長時間ループは不要 = 再起動で処理される)。
    public func post(title: String, subtitle: String?, message: String, pane: String?, sock: String?) {
        _ = ensureSetup()

        let authSema = DispatchSemaphore(value: 0)
        var authorized = false
        center.requestAuthorization(options: [.alert, .sound]) { granted, error in
            authorized = granted
            if let error = error {
                fputs("notification authorization error: \(error.localizedDescription)\n", stderr)
            }
            authSema.signal()
        }
        authSema.wait()

        guard authorized else {
            fputs("notifications not authorized; enable in System Settings > Notifications\n", stderr)
            return
        }

        let content = UNMutableNotificationContent()
        content.title = title
        if let subtitle = subtitle, !subtitle.isEmpty { content.subtitle = subtitle }
        content.body = message
        content.sound = .default
        var info: [String: String] = [:]
        if let pane = pane { info["pane"] = pane }
        if let sock = sock { info["sock"] = sock }
        content.userInfo = info

        // 同一ペインの通知は最新で置き換える (スタックさせない)。ペイン不明時は固定 id。
        let identifier = pane.map { "claude-code-hooks-pane-\($0)" } ?? "claude-code-hooks"
        let request = UNNotificationRequest(identifier: identifier, content: content, trigger: nil)

        let deliverSema = DispatchSemaphore(value: 0)
        center.add(request) { error in
            if let error = error {
                fputs("notification delivery error: \(error.localizedDescription)\n", stderr)
            }
            deliverSema.signal()
        }
        deliverSema.wait()
    }
}

// MARK: - Privates

private extension NotificationSystem {
    func ensureSetup() -> ActionHandler {
        if let handler = self.handler { return handler }

        _ = NSApplication.shared
        NSApp.setActivationPolicy(.accessory)

        let handler = ActionHandler()
        self.handler = handler

        NSApp.delegate = handler

        NSAppleEventManager.shared().setEventHandler(
            handler,
            andSelector: #selector(ActionHandler.handleGetURL(_:withReplyEvent:)),
            forEventClass: AEEventClass(kInternetEventClass),
            andEventID: AEEventID(kAEGetURL)
        )

        NSApp.finishLaunching()

        center.delegate = handler

        return handler
    }

    func runEventLoop(handler: ActionHandler, timeoutSeconds: Double) {
        let timeout = Date(timeIntervalSinceNow: timeoutSeconds)
        while !handler.actionHandled && Date() < timeout {
            if let event = NSApp.nextEvent(
                matching: .any,
                until: Date(timeIntervalSinceNow: 0.1),
                inMode: .default,
                dequeue: true
            ) {
                NSApp.sendEvent(event)
            }
        }
    }

    func waitForActivation(handler: ActionHandler) {
        if let proc = handler.activationProcess, proc.isRunning {
            proc.waitUntilExit()
        }
    }
}
