package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoName     = "xmlui-invoice"
	branchName   = "main"
	appZipURL    = "https://codeload.github.com/jonudell/" + repoName + "/zip/refs/heads/" + branchName
	xmluiRepoZip = "https://api.github.com/repos/xmlui-com/xmlui/zipball/refs/heads/main"
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
		return baseURL + "xmlui-test-server-windows-amd64.zip"
	default:
		return baseURL + "xmlui-test-server-mac-arm.tar.gz"
	}
}

func downloadWithProgress(url, filename string) ([]byte, error) {
	fmt.Printf("Downloading %s...\n", filename)
	fmt.Printf("  From: %s\n", url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if strings.Contains(url, "api.github.com/repos/xmlui-com/xmlui") {
		if token := os.Getenv("PAT_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  Downloaded: %d bytes\n", len(data))
	return data, nil
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
	installDir, _ := os.Getwd()
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
	xmluiZip, err := downloadWithProgress(xmluiRepoZip, "XMLUI repo")
	if err != nil {
		fmt.Println("Failed to download XMLUI source:", err)
		os.Exit(1)
	}
	tmpDir := filepath.Join(installDir, "xmlui-source")
	os.MkdirAll(tmpDir, 0755)
	if err := unzipTo(xmluiZip, tmpDir); err != nil {
		fmt.Println("Failed to extract XMLUI source:", err)
		os.Exit(1)
	}
	var sourceRoot string
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "xmlui-") {
			sourceRoot = filepath.Join(tmpDir, e.Name())
			break
		}
	}
	os.MkdirAll(filepath.Join(installDir, "docs", "pages", "components"), 0755)
	os.MkdirAll(filepath.Join(installDir, "xmlui", "src", "components"), 0755)
	copyDir(filepath.Join(sourceRoot, "docs", "pages", "components"), filepath.Join(installDir, "docs", "pages", "components"))
	copyDir(filepath.Join(sourceRoot, "xmlui", "src", "components"), filepath.Join(installDir, "xmlui", "src", "components"))
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
	var expectedFiles []string
	if runtime.GOOS == "windows" {
		expectedFiles = []string{"xmlui-mcp.exe", "xmlui-mcp-client.exe", "run-mcp-client.bat"}
	} else {
		expectedFiles = []string{"xmlui-mcp", "xmlui-mcp-client", "run-mcp-client.sh"}
	}
	for _, name := range expectedFiles {
		src := filepath.Join(tmpMCP, name)
		dst := filepath.Join(mcpDir, name)
		if err := os.Rename(src, dst); err != nil {
			fmt.Printf("  Skipping %s (not found?): %v\n", name, err)
			continue
		}
		fmt.Printf("  Moved %s to %s\n", name, dst)
		if strings.HasSuffix(name, ".sh") || !strings.HasSuffix(name, ".exe") {
			_ = ensureExecutable(dst)
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
}

func copyDir(src string, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			os.MkdirAll(dstPath, os.ModePerm)
			copyDir(srcPath, dstPath)
		} else {
			in, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			out, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			io.Copy(out, in)
			in.Close()
			out.Close()
		}
	}
	return nil
}
