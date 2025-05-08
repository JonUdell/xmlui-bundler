// xmlui-launcher/main.go
package main

import (
	"archive/tar"
	"archive/zip"
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
	localBaseDir   = "."
)

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

func main() {
	appDir := localBaseDir
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

	// locate the app directory and run start-mac.sh
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

	// copy server binary into app folder
	dstBinPath := filepath.Join(invoiceDir, filepath.Base(binPath))
	if err := copyFile(binPath, dstBinPath); err != nil {
		fmt.Println("Failed to copy server binary into app folder:", err)
		return
	}

	script := filepath.Join(invoiceDir, "start-mac.sh")
	if err := os.Chmod(script, 0755); err != nil {
		fmt.Println("Failed to chmod start-mac.sh:", err)
		return
	}

	fmt.Println("Attempting to run script:")
	fmt.Println("  Path:", script)
	info, err := os.Stat(script)
	if err != nil {
		fmt.Println("  Stat failed:", err)
	} else {
		fmt.Printf("  Exists: true\n  Mode: %v\n", info.Mode())
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


}