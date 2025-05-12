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
	repoName           = "xmlui-invoice"
	defaultDirName     = "xmlui-getting-started"
	branchName         = "main"
	appZipURL          = "https://codeload.github.com/jonudell/" + repoName + "/zip/refs/heads/" + branchName
	xmluiComponentsURL = "https://github.com/jonudell/xmlui-launcher/releases/download/v1.0.0/xmlui-components.zip"
)

func getPlatformSpecificMCPURL() string {
	baseURL := "https://github.com/jonudell/xmlui-mcp/releases/download/v1.0.0/"
	arch := runtime.GOARCH
	switch runtime.GOOS {
	case "darwin":
		if arch == "arm64" {
			return baseURL + "xmlui-mcp-mac-arm.zip"
		}
		return baseURL + "xmlui-mcp-mac-amd.zip"
	case "linux":
		return baseURL + "xmlui-mcp-linux-amd64.zip"
	case "windows":
		return baseURL + "xmlui-mcp-windows-amd64.zip"
	default:
		return baseURL + "xmlui-mcp-mac-arm.zip"
	}
}

func getPlatformSpecificServerURL() string {
	baseURL := "https://github.com/JonUdell/xmlui-test-server/releases/download/v1.0.0/"
	arch := runtime.GOARCH
	switch runtime.GOOS {
	case "darwin":
		if arch == "arm64" {
			return baseURL + "xmlui-test-server-mac-arm.tar.gz"
		}
		return baseURL + "xmlui-test-server-mac-amd.tar.gz"
	case "linux":
		return baseURL + "xmlui-test-server-linux-amd64.tar.gz"
	case "windows":
		return baseURL + "xmlui-test-server-windows64.zip"
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
	for _, f := range r.File {
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

func untarGzTo(data []byte, dest string) error {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fpath := filepath.Join(dest, hdr.Name)
		if hdr.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		out, err := os.Create(fpath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tarReader); err != nil {
			return err
		}
		out.Close()
		if err := ensureExecutable(fpath); err != nil {
			return err
		}
	}
	return nil
}

func ensureExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", path).Run()
	}
	return nil
}

func moveIntoPlace(srcParent, repoName, installDir string) (string, error) {
	repoPrefix := repoName + "-"
	entries, err := os.ReadDir(srcParent)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), repoPrefix) {
			tmp := filepath.Join(srcParent, e.Name())
			final := filepath.Join(installDir, repoName)
			if err := os.Rename(tmp, final); err != nil {
				return "", err
			}
			return final, nil
		}
	}
	return "", fmt.Errorf("repo dir not found")
}

func main() {
	home, _ := os.UserHomeDir()
	installDir := promptForInstallPath(filepath.Join(home, defaultDirName))
	os.MkdirAll(installDir, 0755)

	fmt.Println("Step 1/5: Downloading XMLUI invoice app...")
	appZip, err := downloadWithProgress(appZipURL, "XMLUI invoice app")
	if err != nil {
		fmt.Println("Failed to download app:", err)
		os.Exit(1)
	}
	if err := unzipTo(appZip, installDir); err != nil {
		fmt.Println("Failed to extract app:", err)
		os.Exit(1)
	}

	appDir, err := moveIntoPlace(installDir, repoName, installDir)
	if err != nil {
		fmt.Println("Failed to organize app directory:", err)
		os.Exit(1)
	}

	fmt.Println("Step 2/5: Downloading XMLUI components...")
	components, err := downloadWithProgress(xmluiComponentsURL, "XMLUI components")
	if err != nil {
		fmt.Println("Failed to download components:", err)
		os.Exit(1)
	}
	if err := unzipTo(components, installDir); err != nil {
		fmt.Println("Failed to extract components:", err)
		os.Exit(1)
	}
	fmt.Println("✓ Extracted components")

	srcFrom := filepath.Join(installDir, "xmlui", "src")
	srcTo := filepath.Join(installDir, "src")
	_ = os.Rename(srcFrom, srcTo)
	_ = os.RemoveAll(filepath.Join(installDir, "xmlui"))

	fmt.Println("Step 3/5: Downloading MCP tools...")
	mcpZip, err := downloadWithProgress(getPlatformSpecificMCPURL(), "MCP tools")
	if err != nil {
		fmt.Println("Failed to download MCP tools:", err)
		os.Exit(1)
	}
	tmpMCP := filepath.Join(installDir, "mcpTmp")
	os.MkdirAll(tmpMCP, 0755)
	if err := unzipTo(mcpZip, tmpMCP); err != nil {
		fmt.Println("Failed to extract MCP tools:", err)
		os.Exit(1)
	}

	mcpDir := filepath.Join(installDir, "mcp")
	os.MkdirAll(mcpDir, 0755)
	for _, name := range []string{"xmlui-mcp", "xmlui-mcp-client", "run-mcp-client.sh"} {
		src := filepath.Join(tmpMCP, name)
		dst := filepath.Join(mcpDir, name)
		if err := os.Rename(src, dst); err == nil {
			if strings.HasSuffix(name, ".sh") || !strings.HasSuffix(name, ".exe") {
				_ = ensureExecutable(dst)
			}
		}
	}
	_ = os.RemoveAll(tmpMCP)

	fmt.Println("Step 4/5: Downloading XMLUI test server...")
	serverURL := getPlatformSpecificServerURL()
	serverArchive, err := downloadWithProgress(serverURL, "test server")
	if err != nil {
		fmt.Println("Failed to download server:", err)
		os.Exit(1)
	}
	if strings.HasSuffix(serverURL, ".zip") {
		err = unzipTo(serverArchive, appDir)
	} else {
		err = untarGzTo(serverArchive, appDir)
	}
	if err != nil {
		fmt.Println("Failed to extract server:", err)
		os.Exit(1)
	}

	_ = ensureExecutable(filepath.Join(appDir, "start.sh"))

	fmt.Println("✓ Organized layout complete")
	fmt.Printf("\nInstall location: %s\n", installDir)

	script := filepath.Join(appDir, "start.sh")
	fmt.Println("Launching server:", script)
	cmd := exec.Command(script)
	cmd.Dir = appDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Println("Error launching startup script:", err)
		os.Exit(1)
	}
}
