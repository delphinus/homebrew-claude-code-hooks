// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "claude-code-hooks-notify",
    platforms: [.macOS(.v13)],
    targets: [
        .target(
            name: "ClaudeNotifyLib",
            path: "Sources/ClaudeNotifyLib"
        ),
        .executableTarget(
            name: "claude-code-hooks-notify",
            dependencies: ["ClaudeNotifyLib"],
            path: "Sources/claude-code-hooks-notify"
        ),
        .executableTarget(
            name: "claude-code-hooks-notify-tests",
            dependencies: ["ClaudeNotifyLib"],
            path: "Sources/claude-code-hooks-notify-tests"
        ),
    ],
    swiftLanguageModes: [.v5]
)
