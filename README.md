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
4. Provide instructions for starting the server

### Starting the Server

After installation:

```bash
# macOS/Linux
cd ~/xmlui-getting-started/xmlui-invoice
./start-mac.sh

# Windows
cd "C:\Users\[YourUsername]\xmlui-getting-started\xmlui-invoice"
start-windows.bat
```

## Development

### Building from Source

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Test locally
make test
```

### CI/CD

The project uses GitHub Actions to automatically build binaries for all supported platforms and package the XMLUI components. Releases are automatically created when you create a new release tag.

## Architecture

The launcher downloads the following components:

1. **XMLUI Invoice App**: The example application from `jonudell/xmlui-invoice`
2. **Test Server**: Platform-specific server binary from `JonUdell/xmlui-test-server`
3. **XMLUI Components**: Pre-packaged docs and source components (built by CI)
4. **XMLUI MCP Binary**: Platform-specific MCP binary from `jonudell/xmlui-mcp`

All components are downloaded from public GitHub releases, eliminating the need for authentication tokens.

## License

MIT License - see LICENSE file for details
