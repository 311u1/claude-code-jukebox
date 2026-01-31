import AppKit
import MediaPlayer
import Foundation

// MARK: - go-librespot API types

struct LibrespotStatus: Decodable {
    let stopped: Bool
    let paused: Bool
    let buffering: Bool
    let volume: Int
    let volumeSteps: Int
    let track: TrackInfo?

    enum CodingKeys: String, CodingKey {
        case stopped, paused, buffering, volume, track
        case volumeSteps = "volume_steps"
    }
}

struct TrackInfo: Decodable {
    let uri: String
    let name: String
    let artistNames: [String]
    let albumName: String
    let albumCoverUrl: String?
    let position: Int
    let duration: Int

    enum CodingKeys: String, CodingKey {
        case uri, name, position, duration
        case artistNames = "artist_names"
        case albumName = "album_name"
        case albumCoverUrl = "album_cover_url"
    }
}

// MARK: - Bridge

class NowPlayingBridge {
    let baseURL: String
    let infoCenter = MPNowPlayingInfoCenter.default()
    let commandCenter = MPRemoteCommandCenter.shared()
    var pollTimer: Timer?
    var lastTrackUri: String?
    var cachedArtwork: MPMediaItemArtwork?
    var cachedArtworkURL: String?

    init(baseURL: String = "http://localhost:3678") {
        self.baseURL = baseURL
    }

    func start() {
        registerCommands()
        // Poll every 2 seconds
        pollTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            self?.poll()
        }
        // Initial poll
        poll()
        print("NowPlayingBridge: running (polling \(baseURL))")
    }

    // MARK: - Remote command registration

    func registerCommands() {
        commandCenter.togglePlayPauseCommand.isEnabled = true
        commandCenter.togglePlayPauseCommand.addTarget { [weak self] _ in
            self?.sendCommand("playpause")
            return .success
        }

        commandCenter.playCommand.isEnabled = true
        commandCenter.playCommand.addTarget { [weak self] _ in
            self?.sendCommand("resume")
            return .success
        }

        commandCenter.pauseCommand.isEnabled = true
        commandCenter.pauseCommand.addTarget { [weak self] _ in
            self?.sendCommand("pause")
            return .success
        }

        commandCenter.nextTrackCommand.isEnabled = true
        commandCenter.nextTrackCommand.addTarget { [weak self] _ in
            self?.sendCommand("next")
            return .success
        }

        commandCenter.previousTrackCommand.isEnabled = true
        commandCenter.previousTrackCommand.addTarget { [weak self] _ in
            self?.sendCommand("prev")
            return .success
        }

        commandCenter.changePlaybackPositionCommand.isEnabled = true
        commandCenter.changePlaybackPositionCommand.addTarget { [weak self] event in
            guard let posEvent = event as? MPChangePlaybackPositionCommandEvent else {
                return .commandFailed
            }
            self?.seekTo(positionMs: Int(posEvent.positionTime * 1000))
            return .success
        }
    }

    // MARK: - HTTP helpers

    func sendCommand(_ command: String) {
        guard let url = URL(string: "\(baseURL)/player/\(command)") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 3
        URLSession.shared.dataTask(with: request).resume()
    }

    func seekTo(positionMs: Int) {
        guard let url = URL(string: "\(baseURL)/player/seek") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONEncoder().encode(["position": positionMs])
        request.timeoutInterval = 3
        URLSession.shared.dataTask(with: request).resume()
    }

    // MARK: - Polling

    func poll() {
        guard let url = URL(string: "\(baseURL)/status") else { return }
        var request = URLRequest(url: url)
        request.timeoutInterval = 3

        URLSession.shared.dataTask(with: request) { [weak self] data, _, error in
            guard let self = self, let data = data, error == nil else {
                // go-librespot not running — clear now playing
                DispatchQueue.main.async {
                    self?.clearNowPlaying()
                }
                return
            }

            guard let status = try? JSONDecoder().decode(LibrespotStatus.self, from: data) else { return }

            DispatchQueue.main.async {
                self.updateNowPlaying(status: status)
            }
        }.resume()
    }

    func clearNowPlaying() {
        infoCenter.nowPlayingInfo = nil
        infoCenter.playbackState = .stopped
    }

    func updateNowPlaying(status: LibrespotStatus) {
        if status.stopped {
            infoCenter.playbackState = .stopped
            infoCenter.nowPlayingInfo = nil
            lastTrackUri = nil
            return
        }

        guard let track = status.track else {
            if status.buffering {
                infoCenter.playbackState = .interrupted
            }
            return
        }

        // Update playback state
        if status.paused {
            infoCenter.playbackState = .paused
        } else if status.buffering {
            infoCenter.playbackState = .interrupted
        } else {
            infoCenter.playbackState = .playing
        }

        // Build now playing info
        var info: [String: Any] = [
            MPMediaItemPropertyTitle: track.name,
            MPMediaItemPropertyArtist: track.artistNames.joined(separator: ", "),
            MPMediaItemPropertyAlbumTitle: track.albumName,
            MPMediaItemPropertyPlaybackDuration: Double(track.duration) / 1000.0,
            MPNowPlayingInfoPropertyElapsedPlaybackTime: Double(track.position) / 1000.0,
            MPNowPlayingInfoPropertyPlaybackRate: status.paused ? 0.0 : 1.0,
            MPNowPlayingInfoPropertyMediaType: MPNowPlayingInfoMediaType.audio.rawValue,
        ]

        // Fetch album art if track changed
        if track.uri != lastTrackUri, let artURL = track.albumCoverUrl {
            lastTrackUri = track.uri
            fetchArtwork(urlString: artURL) { [weak self] artwork in
                guard let self = self, let artwork = artwork else { return }
                DispatchQueue.main.async {
                    self.cachedArtwork = artwork
                    self.cachedArtworkURL = artURL
                    if var currentInfo = self.infoCenter.nowPlayingInfo {
                        currentInfo[MPMediaItemPropertyArtwork] = artwork
                        self.infoCenter.nowPlayingInfo = currentInfo
                    }
                }
            }
        }

        if let artwork = cachedArtwork {
            info[MPMediaItemPropertyArtwork] = artwork
        }

        infoCenter.nowPlayingInfo = info
    }

    func fetchArtwork(urlString: String, completion: @escaping (MPMediaItemArtwork?) -> Void) {
        guard let url = URL(string: urlString) else {
            completion(nil)
            return
        }

        URLSession.shared.dataTask(with: url) { data, _, _ in
            guard let data = data, let image = NSImage(data: data) else {
                completion(nil)
                return
            }

            let artwork = MPMediaItemArtwork(boundsSize: image.size) { _ in image }
            completion(artwork)
        }.resume()
    }
}

// MARK: - Main

// We need NSApplication for MPRemoteCommandCenter to work on macOS.
// It won't receive media key events without an app run loop.
let app = NSApplication.shared
app.setActivationPolicy(.accessory) // No dock icon, no menu bar

let bridge = NowPlayingBridge()
bridge.start()

app.run()
