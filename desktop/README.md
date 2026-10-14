# New API Desktop

The desktop app of New API, built with [MyGo](https://github.com/egoist/mygo).
It runs the `new-api` server it ships with on port 3000 and shows the web
console in the system webview (WebView2 on Windows, WKWebView on macOS,
WebKitGTK on Linux). Closing the window keeps the server running in the tray;
choose Quit in the tray menu to stop it.

This directory is a Go module of its own, so that MyGo's Go version (1.27.1 or
later) does not apply to the gateway.

## Requirements

- Go 1.27.1 or later, and Bun, which builds the frontend
- To make the Windows installer on Linux or macOS: NSIS (`apt install nsis`,
  `brew install makensis`)
- To run the app: the WebView2 Runtime on Windows 10 (Windows 11 has it);
  GTK 3, WebKitGTK 4.1 and `libayatana-appindicator3` (for the tray) on Linux

## Build

```bash
# The frontend, the server and the app for this machine
./build.sh

# Other platforms: MyGo needs no cgo, so any machine builds them
./build.sh windows/amd64
./build.sh windows/amd64,linux/amd64,darwin/universal

# Reuse web/dist from an earlier build
SKIP_WEB=1 ./build.sh windows/amd64
```

`build.sh` cross-compiles the server into `resources/<os>-<arch>/bin/`, which
MyGo ships with the app of that platform, then runs `go tool mygo build`. The
output is in `build/<os>-<arch>/`:

| Platform | Output |
| --- | --- |
| Windows | `New API.exe` with its files, and the installer `New API Setup <version> amd64.exe` |
| macOS | `New API.app` and a disk image |
| Linux | the executable with its desktop entry, a `.tar.gz` with `install.sh`, and a `.deb` |

## Development

Start the backend and the frontend from the repository root, then the app:

```bash
go run main.go     # backend, port 3000
make dev-web       # frontend dev server, port 5173

cd desktop && go tool mygo dev
```

Development builds load the frontend dev server and do not start a server of
their own.

## Data and logs

| | Windows | macOS | Linux |
| --- | --- | --- | --- |
| Database (`data/new-api.db`) | `%AppData%\New API\` | `~/Library/Application Support/New API/` | `~/.config/New API/` |
| Logs (`server.log` and the server's own) | `%AppData%\New API\logs\` | `~/Library/Logs/New API/` | `~/.config/New API/logs/` |

An existing database of the former Electron app, in `new-api-electron/data`
under the same parent directory, keeps being used while the directory above
has none. The server runs in the data directory, so a `.env` file there sets
its environment.
