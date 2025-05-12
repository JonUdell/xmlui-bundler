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
	xmluiRepoZip = "https://codeload.github.com/xmlui-com/xmlui/zip/refs/heads/main"
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

	if strings.Contains(url, "codeload.github.com/xmlui-com/xmlui") {
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			fmt.Println("  Using authentication token for private repository")
			req.SetBasicAuth(token, "x-oauth-basic")
		} else {
			fmt.Println("  Warning: No authentication token found for private repository")
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if strings.Contains(url, "codeload.github.com/xmlui-com/xmlui") && resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("authentication failed for private repository: %s (status: %s) - check PAT_TOKEN", url, resp.Status)
		}
		return nil, fmt.Errorf("request failed: %s for URL: %s", resp.Status, url)
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
		cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
		cmd.Run() // Ignore errors, as file might not have quarantine attribute
		
		// Verify quarantine was removed or never existed
		verifyCmd := exec.Command("xattr", "-l", path)
		output, _ := verifyCmd.CombinedOutput()
		if strings.Contains(string(output), "com.apple.quarantine") {
			fmt.Printf("  Warning: Quarantine could not be removed from %s\n", path)
		} else {
			fmt.Printf("  Quarantine removed or not present on %s\n", path)
		}
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
	// Clean up the source directory - we don't need it
	_ = os.RemoveAll(tmpDir)

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
	
	// First ensure docs and src directories are created under mcp
	docsDir := filepath.Join(mcpDir, "docs")
	srcDir := filepath.Join(mcpDir, "src")
	os.MkdirAll(docsDir, 0755)
	os.MkdirAll(srcDir, 0755)
	
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
	
	// Clean up the temporary MCP directory
	_ = os.RemoveAll(tmpMCP)
	
	// Move docs and src under mcp if they exist at the root level
	if _, err := os.Stat(filepath.Join(installDir, "docs")); err == nil {
		if err := os.Rename(filepath.Join(installDir, "docs"), docsDir); err != nil {
			fmt.Printf("Warning: Could not move docs directory: %v\n", err)
		}
	}
	
	if _, err := os.Stat(filepath.Join(installDir, "src")); err == nil {
		if err := os.Rename(filepath.Join(installDir, "src"), srcDir); err != nil {
			fmt.Printf("Warning: Could not move src directory: %v\n", err)
		}
	}

	// Ensure all executables in mcp directory have proper permissions
	files, _ := os.ReadDir(mcpDir)
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(mcpDir, file.Name())
			// Skip extensions that are not executables
			if strings.HasSuffix(file.Name(), ".exe") || 
			   strings.HasSuffix(file.Name(), ".sh") || 
			   !strings.Contains(file.Name(), ".") {
				ensureExecutable(filePath)
			}
		}
	}

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

	// The final bundle should contain only these files/directories:
	// - xmlui-invoice/  (the invoice app)
	// - mcp/  (with docs/ and src/ inside it)
	// - XMLUI_GETTING_STARTED_README.md
	
	// Write a cleanup script that will remove files not in the include list
	if runtime.GOOS == "windows" {
		cleanupScript := "@echo off\r\n"
		cleanupScript += "echo Cleaning up temporary files...\r\n"
		cleanupScript += fmt.Sprintf("if exist \"%s\" del \"%s\"\r\n", filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
		cleanupScript += "if exist *.zip del *.zip\r\n"
		cleanupScript += "del cleanup.bat\r\n"
		os.WriteFile(filepath.Join(installDir, "cleanup.bat"), []byte(cleanupScript), 0755)
		fmt.Println("Note: Run cleanup.bat to remove the bundler executable and temporary files")
	} else {
		cleanupScript := "#!/bin/sh\n"
		cleanupScript += "echo Cleaning up temporary files...\n"
		cleanupScript += fmt.Sprintf("rm -f \"%s\"\n", filepath.Base(os.Args[0]))
		cleanupScript += "rm -f *.zip\n"
		cleanupScript += "rm -f cleanup.sh\n"
		os.WriteFile(filepath.Join(installDir, "cleanup.sh"), []byte(cleanupScript), 0755)
		ensureExecutable(filepath.Join(installDir, "cleanup.sh"))
		fmt.Println("Note: Run ./cleanup.sh to remove the bundler executable and temporary files")
	}

	fmt.Println("✓ Organized layout complete")
	fmt.Printf("\nInstall location: %s\n", installDir)
}


