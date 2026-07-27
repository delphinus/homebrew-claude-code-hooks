import AppKit
import UserNotifications
import Foundation

final class ActionHandler: NSObject {
    var actionHandled = false
    var activationProcess: Process?

    private func activate(pane: String?, sock: String?) {
        guard let pane = pane else { return }
        activationProcess = Activator.run(pane: pane, sock: sock)
    }

    // MARK: URL scheme (open "claude-code-hooks://activate?pane=...")

    @objc func handleGetURL(_ event: NSAppleEventDescriptor, withReplyEvent reply: NSAppleEventDescriptor) {
        guard let urlString = event.paramDescriptor(forKeyword: AEKeyword(keyDirectObject))?.stringValue,
              let url = URL(string: urlString) else {
            return
        }
        handleURL(url)
    }

    private func handleURL(_ url: URL) {
        guard let params = Activator.parse(url: url) else { return }
        activate(pane: params.pane, sock: params.sock)
        actionHandled = true
    }
}

// MARK: - NSApplicationDelegate

extension ActionHandler: NSApplicationDelegate {
    func application(_ application: NSApplication, open urls: [URL]) {
        for url in urls {
            handleURL(url)
        }
    }
}

// MARK: - UNUserNotificationCenterDelegate

extension ActionHandler: UNUserNotificationCenterDelegate {
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        // 通知本体のタップ (default action) のみを扱う。
        if response.actionIdentifier == UNNotificationDefaultActionIdentifier {
            let info = response.notification.request.content.userInfo
            activate(pane: info["pane"] as? String, sock: info["sock"] as? String)
        }
        actionHandled = true
        completionHandler()
    }
}
