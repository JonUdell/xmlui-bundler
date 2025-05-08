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
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	appZipURL      = "https://codeload.github.com/jonudell/xmlui-invoice/zip/refs/heads/main"
	serverTarGzURL = "https://github.com/JonUdell/xmlui-test-server/releases/download/v1.0.0/xmlui-test-server-mac-arm.tar.gz"
)

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

func askUserForFolder(defaultPath string) string {
	if term.IsTerminal(int(syscall.Stdin)) {
		return promptForInstallPath(defaultPath)
	}

	script := `tell application "System Events"
	activate
	set chosenFolder to choose folder with prompt "Choose install location for the app"
	set posixPath to POSIX path of chosenFolder
end tell
return posixPath`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		fmt.Println("⚠️  Folder selection cancelled or failed.")
		fmt.Println("→ Installing to default location:", defaultPath)

		_ = exec.Command("osascript", "-e",
		fmt.Sprintf(`display dialog "No folder was selected. The app will be installed to: %s" with title "Install Location Fallback" buttons {"OK"}`, escapeAppleScriptString(defaultPath))).Run()
		return defaultPath
	}

	return strings.TrimSpace(string(out))
}

func downloadWithCurl(url string) ([]byte, error) {
	fmt.Println("Downloading with curl:", url)
	cmd := exec.Command("curl", "-L", "-sS", url)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("curl failed: %w", err)
	}
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

func untarGzTo(data []byte, dest string) (string, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	var lastBinaryPath string
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
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

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func escapeAppleScriptString(s string) string {
	return strings.ReplaceAll(s, "\"", `\"`)
}

func ensureExecutable(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if err := os.Chmod(path, 0755); err != nil {
		fmt.Printf("Go chmod failed for %s: %v\n", path, err)
	} else {
		fmt.Printf("Go chmod succeeded for %s\n", path)
	}
	if err := exec.Command("chmod", "+x", path).Run(); err != nil {
		fmt.Printf("Shell chmod +x failed for %s: %v\n", path, err)
	} else {
		fmt.Printf("Shell chmod +x succeeded for %s\n", path)
	}
	if fi, err := os.Stat(path); err == nil {
		fmt.Printf("Final mode for %s: %v\n", path, fi.Mode())
	}
	return nil
}

func main() {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home)
	installDir := askUserForFolder(defaultDir)
	os.MkdirAll(installDir, 0755)

	appZip, err := downloadWithCurl(appZipURL)
	if err != nil {
		fmt.Println("Failed to download app zip:", err)
		return
	}
	if err := unzipTo(appZip, installDir); err != nil {
		fmt.Println("Failed to unzip app:", err)
		return
	}

	serverTarGz, err := downloadWithCurl(serverTarGzURL)
	if err != nil {
		fmt.Println("Failed to download server tar.gz:", err)
		return
	}
	binPath, err := untarGzTo(serverTarGz, installDir)
	if err != nil {
		fmt.Println("Failed to unpack server tar.gz:", err)
		return
	}

	dirs, err := os.ReadDir(installDir)
	if err != nil {
		fmt.Println("Failed to read install directory:", err)
		return
	}

	var appDir string
	for _, d := range dirs {
		if d.IsDir() && strings.HasSuffix(d.Name(), "-main") {
			appDir = filepath.Join(installDir, strings.TrimSuffix(d.Name(), "-main"))
			err := os.Rename(filepath.Join(installDir, d.Name()), appDir)
			if err != nil {
				fmt.Println("Failed to rename extracted app folder:", err)
				return
			}
			break
		}
	}

	if appDir == "" {
		fmt.Println("Failed to locate extracted app folder")
		return
	}
	dstBinPath := filepath.Join(appDir, filepath.Base(binPath))
	if err := os.Rename(binPath, dstBinPath); err != nil {
		fmt.Println("Failed to move server binary into app folder:", err)
		return
	}

	script := filepath.Join(appDir, "start-mac.sh")
	if err := ensureExecutable(script); err != nil {
		fmt.Println("Failed to make start-mac.sh executable:", err)
		return
	}

	cmd := exec.Command("osascript", "-e",
		fmt.Sprintf(`tell application "Terminal" to do script "cd \"%s\" && bash -i -c './start-mac.sh; echo; echo Server exited. Type exit to close this window.'"`, appDir),
		"-e", `tell application "Terminal" to activate`)
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to launch start-mac.sh in Terminal:", err)
		return
	}

	fmt.Println("✅ App launched.")
}
