// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "NowPlayingBridge",
    platforms: [.macOS(.v12)],
    targets: [
        .executableTarget(
            name: "NowPlayingBridge",
            path: "Sources",
            linkerSettings: [
                .linkedFramework("MediaPlayer"),
                .linkedFramework("AppKit"),
            ]
        ),
    ]
)
