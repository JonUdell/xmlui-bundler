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

func ensureExecutable(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	// Try Go chmod first
	if err := os.Chmod(path, 0755); err != nil {
		fmt.Printf("Go chmod failed for %s: %v\n", path, err)
	} else {
		fmt.Printf("Go chmod succeeded for %s\n", path)
	}

	// Fallback to shell chmod
	if err := exec.Command("chmod", "+x", path).Run(); err != nil {
		fmt.Printf("Shell chmod +x failed for %s: %v\n", path, err)
	} else {
		fmt.Printf("Shell chmod +x succeeded for %s\n", path)
	}

	// Print final mode
	if fi, err := os.Stat(path); err == nil {
		fmt.Printf("Final mode for %s: %v\n", path, fi.Mode())
	}
	return nil
}


func main() {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home)
	appDir := promptForInstallPath(defaultDir)
	os.MkdirAll(appDir, 0755)

	appZip, err := downloadWithCurl(appZipURL)
	if err != nil {
		fmt.Println("Failed to download app zip:", err)
		return
	}
	if err := unzipTo(appZip, appDir); err != nil {
		fmt.Println("Failed to unzip app:", err)
		return
	}

	serverTarGz, err := downloadWithCurl(serverTarGzURL)
	if err != nil {
		fmt.Println("Failed to download server tar.gz:", err)
		return
	}
	binPath, err := untarGzTo(serverTarGz, appDir)
	if err != nil {
		fmt.Println("Failed to unpack server tar.gz:", err)
		return
	}

	dirs, err := os.ReadDir(appDir)
	if err != nil {
		fmt.Println("Failed to read app directory:", err)
		return
	}

	var invoiceDir string
	for _, d := range dirs {
		if d.IsDir() && strings.HasPrefix(d.Name(), "xmlui-invoice") {
			invoiceDir = filepath.Join(appDir, d.Name())
			break
		}
	}

	if invoiceDir == "" {
		fmt.Println("Failed to locate extracted invoice app folder")
		return
	}

	dstBinPath := filepath.Join(invoiceDir, filepath.Base(binPath))
	if err := copyFile(binPath, dstBinPath); err != nil {
		fmt.Println("Failed to copy server binary into app folder:", err)
		return
	}


	script := filepath.Join(invoiceDir, "start-mac.sh")
	if err := ensureExecutable(script); err != nil {
		fmt.Println("Failed to make start-mac.sh executable:", err)
		return
	}

	absScript, err := filepath.Abs(script)
	if err != nil {
		fmt.Println("Failed to resolve absolute script path:", err)
		return
	}

	cmd := exec.Command("/bin/bash", absScript)
	cmd.Dir = invoiceDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to launch start-mac.sh:", err)
		return
	}

	fmt.Println("All done.")
}
