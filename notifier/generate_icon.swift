import AppKit

// claude-code-hooks-notify のアプリアイコンを生成する。
// チャコールの squircle に、ターミナルのプロンプト ">" とブロックカーソルを
// Claude 系の暖色 (coral) で描く。SF Symbol に依存せず CoreGraphics で描画する
// ので、全サイズでシャープに出る。

let sizes: [(name: String, size: Int)] = [
    ("icon_16x16", 16),
    ("icon_16x16@2x", 32),
    ("icon_32x32", 32),
    ("icon_32x32@2x", 64),
    ("icon_128x128", 128),
    ("icon_128x128@2x", 256),
    ("icon_256x256", 256),
    ("icon_256x256@2x", 512),
    ("icon_512x512", 512),
    ("icon_512x512@2x", 1024),
]

let iconsetPath = "AppIcon.iconset"
let fm = FileManager.default
try? fm.removeItem(atPath: iconsetPath)
try! fm.createDirectory(atPath: iconsetPath, withIntermediateDirectories: true)

// coral (Claude 系の暖色)
let coral = CGColor(red: 0.902, green: 0.482, blue: 0.353, alpha: 1.0)

for entry in sizes {
    let s = CGFloat(entry.size)
    let image = NSImage(size: NSSize(width: s, height: s))
    image.lockFocus()
    let ctx = NSGraphicsContext.current!.cgContext

    // 背景: チャコールの squircle + 縦グラデーション
    let radius = s * 0.2237
    let bgRect = CGRect(x: 0, y: 0, width: s, height: s)
    let bgPath = CGPath(roundedRect: bgRect, cornerWidth: radius, cornerHeight: radius, transform: nil)
    ctx.saveGState()
    ctx.addPath(bgPath)
    ctx.clip()

    let colorSpace = CGColorSpaceCreateDeviceRGB()
    let bgColors = [
        CGColor(red: 0.216, green: 0.216, blue: 0.235, alpha: 1.0), // 上 (明るめ)
        CGColor(red: 0.106, green: 0.106, blue: 0.125, alpha: 1.0), // 下 (暗め)
    ] as CFArray
    let bgGradient = CGGradient(colorsSpace: colorSpace, colors: bgColors, locations: [0.0, 1.0])!
    ctx.drawLinearGradient(bgGradient, start: CGPoint(x: 0, y: s), end: CGPoint(x: 0, y: 0), options: [])

    // 上端にごく淡いハイライト (立体感)
    let glossColors = [
        CGColor(red: 1.0, green: 1.0, blue: 1.0, alpha: 0.10),
        CGColor(red: 1.0, green: 1.0, blue: 1.0, alpha: 0.0),
    ] as CFArray
    let gloss = CGGradient(colorsSpace: colorSpace, colors: glossColors, locations: [0.0, 1.0])!
    ctx.drawLinearGradient(gloss, start: CGPoint(x: 0, y: s), end: CGPoint(x: 0, y: s * 0.62), options: [])
    ctx.restoreGState()

    // プロンプト ">" (coral, 丸端の折れ線)
    ctx.saveGState()
    ctx.setStrokeColor(coral)
    ctx.setLineWidth(s * 0.075)
    ctx.setLineCap(.round)
    ctx.setLineJoin(.round)
    ctx.move(to: CGPoint(x: s * 0.29, y: s * 0.64))
    ctx.addLine(to: CGPoint(x: s * 0.45, y: s * 0.50))
    ctx.addLine(to: CGPoint(x: s * 0.29, y: s * 0.36))
    ctx.strokePath()

    // ブロックカーソル (coral, 角丸矩形)
    let cursorRect = CGRect(x: s * 0.53, y: s * 0.355, width: s * 0.16, height: s * 0.29)
    let cursorPath = CGPath(
        roundedRect: cursorRect,
        cornerWidth: s * 0.035,
        cornerHeight: s * 0.035,
        transform: nil
    )
    ctx.addPath(cursorPath)
    ctx.setFillColor(coral)
    ctx.fillPath()
    ctx.restoreGState()

    image.unlockFocus()

    guard let tiffData = image.tiffRepresentation,
          let bitmap = NSBitmapImageRep(data: tiffData),
          let pngData = bitmap.representation(using: .png, properties: [:]) else {
        fputs("Failed to create PNG for \(entry.name)\n", stderr)
        continue
    }
    try! pngData.write(to: URL(fileURLWithPath: "\(iconsetPath)/\(entry.name).png"))
}

print("Iconset created at \(iconsetPath)")
