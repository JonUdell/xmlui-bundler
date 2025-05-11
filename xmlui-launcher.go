// xmlui-launcher/main.go
package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoName       = "xmlui-invoice"
	defaultDirName = "xmlui-getting-started"
	branchName     = "main"
	appZipURL      = "https://codeload.github.com/jonudell/" + repoName + "/zip/refs/heads/" + branchName
	serverTarGzURL = "https://github.com/JonUdell/xmlui-test-server/releases/download/v1.0.0/xmlui-test-server-mac-arm.tar.gz"
	// Pre-built XMLUI components from public release
	xmluiComponentsURL = "https://github.com/jonudell/xmlui-launcher/releases/download/v1.0.0/xmlui-components.zip"
)



func getPlatformSpecificMCPURL() string {
	baseURL := "https://github.com/jonudell/xmlui-mcp/releases/download/v1.0.0/"

	// Determine platform and architecture
	arch := runtime.GOARCH

	switch runtime.GOOS {
	case "darwin":
		if arch == "arm64" {
			return baseURL + "xmlui-mcp-mac-arm.zip"
		} else {
			return baseURL + "xmlui-mcp-mac-amd.zip"
		}
	case "linux":
		return baseURL + "xmlui-mcp-linux-amd.zip"
	case "windows":
		return baseURL + "xmlui-mcp-windows.zip"
	default:
		return baseURL + "xmlui-mcp-mac-arm.zip"
	}

}

func getPlatformSpecificServerURL() string {
	baseURL := "https://github.com/JonUdell/xmlui-test-server/releases/download/v1.0.0/"

	// Determine platform and architecture
	arch := runtime.GOARCH

	switch runtime.GOOS {
	case "darwin":
		if arch == "arm64" {
			return baseURL + "xmlui-test-server-mac-arm.tar.gz"
		} else {
			return baseURL + "xmlui-test-server-mac-amd.tar.gz"
		}
	case "linux":
		return baseURL + "xmlui-test-server-linux-amd.tar.gz"
	case "windows":
		return baseURL + "xmlui-test-server-windows.zip"
	default:
		return baseURL + "xmlui-test-server-mac-arm.tar.gz"
	}

}

func promptForInstallPath(defaultPath string) string {
	fmt.Printf("Install app to default location (%s)? [Y/n]: ", defaultPath)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if strings.ToLower(input) == "n" {
		fmt.Print("Enter custom install path: ")
		scanner.Scan()
		customPath := strings.TrimSpace(scanner.Text())
		if customPath != "" {
			return customPath
		}
	}
	return defaultPath
}

func downloadWithProgress(url, filename string) ([]byte, error) {
	fmt.Printf("Downloading %s...\n", filename)
	fmt.Printf("  From: %s\n", url)

	cmd := exec.Command("curl", "-L", "-sS", "-#", url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("curl failed: %w", err)
	}

	fmt.Printf("  Downloaded: %d bytes\n", out.Len())
	return out.Bytes(), nil
}

func unzipTo(data []byte, dest string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	fmt.Printf("  Extracting %d files...\n", len(r.File))
	count := 0
	for _, f := range r.File {
		count++
		if count%10 == 0 || count == len(r.File) {
			fmt.Printf("  Progress: %d/%d files\n", count, len(r.File))
		}

		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(fpath)
		if err != nil {
			return err
		}
		io.Copy(out, in)
		in.Close()
		out.Close()
	}
	return nil
}

func untarGzTo(data []byte, dest string) (string, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	var lastBinaryPath string
	count := 0
	fmt.Println("  Extracting .tar.gz...")

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		count++
		if count%5 == 0 {
			fmt.Printf("  Progress: %d files\n", count)
		}

		fpath := filepath.Join(dest, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		} else {
			os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
			outFile, err := os.Create(fpath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return "", err
			}
			outFile.Chmod(0755)
			outFile.Close()
			lastBinaryPath = fpath
		}
	}
	return lastBinaryPath, nil
}

func ensureExecutable(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if err := os.Chmod(path, 0755); err != nil {
		fmt.Printf("Go chmod failed for %s: %v\n", path, err)
	} else {
		fmt.Printf("Made %s executable\n", path)
	}
	if err := exec.Command("chmod", "+x", path).Run(); err != nil {
		fmt.Printf("Shell chmod +x failed for %s: %v\n", path, err)
	}
	return nil
}

func main() {
	fmt.Println("=== XMLUI Getting Started Installer ===")
	fmt.Printf("Platform: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, defaultDirName)
	installDir := promptForInstallPath(defaultDir)
	os.MkdirAll(installDir, 0755)

	fmt.Printf("\nInstalling to: %s\n", installDir)
	fmt.Println("\n----------------------------------------")

	// Download XMLUI invoice app
	fmt.Println("Step 1/4: Downloading XMLUI invoice app...")
	appZip, err := downloadWithProgress(appZipURL, "XMLUI invoice app")
	if err != nil {
		fmt.Printf("Failed to download app: %v\n", err)
		os.Exit(1)
	}
	if err := unzipTo(appZip, installDir); err != nil {
		fmt.Printf("Failed to extract app: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Complete")

	// Download test server
	fmt.Println("\nStep 2/4: Downloading test server...")
	serverURL := getPlatformSpecificServerURL()
	serverData, err := downloadWithProgress(serverURL, "test server")
	if err != nil {
		fmt.Printf("Failed to download server: %v\n", err)
		os.Exit(1)
	}

	var binPath string
	if strings.HasSuffix(serverURL, ".tar.gz") {
		binPath, err = untarGzTo(serverData, installDir)
	} else {
		// Windows zip file
		if err := unzipTo(serverData, installDir); err != nil {
			fmt.Printf("Failed to extract server: %v\n", err)
			os.Exit(1)
		}
		// Find the executable
		binPath = filepath.Join(installDir, "xmlui-test-server.exe")
	}

	if err != nil {
		fmt.Printf("Failed to extract server: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Complete")

	// Download XMLUI components
	fmt.Println("\nStep 3/4: Downloading XMLUI components...")
	xmluiComponents, err := downloadWithProgress(xmluiComponentsURL, "XMLUI components")
	if err != nil {
		fmt.Printf("Failed to download components: %v\n", err)
		os.Exit(1)
	}
	if err := unzipTo(xmluiComponents, installDir); err != nil {
		fmt.Printf("Failed to extract components: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Complete")

	// Download XMLUI MCP binary
	fmt.Println("\nStep 4/4: Downloading XMLUI MCP binary...")
	mcpURL := getPlatformSpecificMCPURL()
	mcpZip, err := downloadWithProgress(mcpURL, "XMLUI MCP binary")
	if err != nil {
		fmt.Printf("Failed to download MCP binary: %v\n", err)
		os.Exit(1)
	}
	if err := unzipTo(mcpZip, installDir); err != nil {
		fmt.Printf("Failed to extract MCP binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Complete")

	// Organize files
	fmt.Println("\nOrganizing files...")
	dirs, err := os.ReadDir(installDir)
	if err != nil {
		fmt.Printf("Failed to read install directory: %v\n", err)
		os.Exit(1)
	}

	var appDir string
	for _, d := range dirs {
		if d.IsDir() && strings.HasSuffix(d.Name(), "-main") {
			src := filepath.Join(installDir, d.Name())
			dst := filepath.Join(installDir, repoName)
			if err := os.Rename(src, dst); err != nil {
				fmt.Printf("Warning: could not rename %s to %s: %v\n", src, dst, err)
				appDir = src // fallback to unrenamed directory
			} else {
				appDir = dst
			}
			break
		}
	}

	if appDir == "" {
		fmt.Println("Failed to locate extracted app folder")
		os.Exit(1)
	}

	// Move server binary into app folder
	dstBinPath := filepath.Join(appDir, filepath.Base(binPath))
	if err := os.Rename(binPath, dstBinPath); err != nil {
		fmt.Printf("Failed to move server binary: %v\n", err)
		os.Exit(1)
	}

	// Make start script executable (non-Windows)
	if runtime.GOOS != "windows" {
		script := filepath.Join(appDir, "start.sh")
		if err := ensureExecutable(script); err != nil {
			fmt.Printf("Failed to make start script executable: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("✅ Installation Complete!")
	fmt.Println("========================================")

	// Launch the app via platform-specific startup script
	var script string
	if runtime.GOOS == "windows" {
		script = filepath.Join(appDir, "start.bat")
	} else {
		script = filepath.Join(appDir, "start.sh")
	}

	fmt.Printf("Launching: %s\n", script)
	cmd := exec.Command(script)
	cmd.Dir = appDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error launching startup script: %v\n", err)
		os.Exit(1)
	}
}