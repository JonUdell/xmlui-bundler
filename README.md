# XMLUI Launcher

A cross-platform installer for XMLUI that downloads and sets up all necessary components for getting started with XMLUI development.

## Features

- Downloads and installs XMLUI invoice example app
- Downloads and installs XMLUI test server (platform-specific)
- Downloads and installs XMLUI components (docs and source)
- Downloads and installs XMLUI MCP binary (platform-specific)
- Pure CLI interface - no GUI dependencies
- Cross-platform support (macOS, Linux, Windows)
- No authentication required - all downloads from public releases

## Platforms

- **macOS**: ARM64 (Apple Silicon) and AMD64 (Intel)
- **Linux**: AMD64
- **Windows**: AMD64

## Usage

### Download from Release

Download the appropriate binary for your platform from the [releases page](https://github.com/jonudell/xmlui-launcher/releases).

### Run the Installer

```bash
# macOS/Linux
./xmlui-launcher-[platform]

# Windows
xmlui-launcher-windows-amd64.exe
```

The installer will:
1. Ask for an installation directory (defaults to `~/xmlui-getting-started`)
2. Download all necessary components
3. Extract and organize the files
4. Start the XMLUI Invoice app

