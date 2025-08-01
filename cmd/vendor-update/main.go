package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const vendorDir = "static/vendor"

var npmPackages = []struct {
	name       string
	cdnJs      string
	cdnCss     string
	minSizeJs  int64
	minSizeCss int64
}{
	{
		name:      "marked",
		cdnJs:     "https://cdn.jsdelivr.net/npm/marked@%s/lib/marked.umd.js",
		minSizeJs: 10000,
	},
	{
		name:      "marked-highlight",
		cdnJs:     "https://cdn.jsdelivr.net/npm/marked-highlight@%s/lib/index.umd.js",
		minSizeJs: 1000,
	},
	{
		name:       "highlight.js",
		cdnJs:      "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/%s/highlight.min.js",
		cdnCss:     "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/%s/styles/github.min.css",
		minSizeJs:  50000,
		minSizeCss: 500,
	},
}

func getLatestVersion(pkg string) (string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkg)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	var data struct {
		DistTags struct {
			Latest string `json:"latest"`
		} `json:"dist-tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.DistTags.Latest, nil
}

func downloadFile(url, outPath string, minSize int64) error {
	fmt.Printf("Downloading %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	if written < minSize {
		return fmt.Errorf("file %s too small (%d bytes)", outPath, written)
	}
	fmt.Printf("Saved to %s (%d bytes)\n", outPath, written)
	return nil
}

func writeVersionsTxt(versions map[string]string) error {
	path := filepath.Join(vendorDir, "versions.txt")
	content := fmt.Sprintf(`# Vendor Library Versions
# This file tracks the versions of locally stored vendor libraries

marked.js=%s
marked-highlight=%s
highlight.js=%s

# Update URLs
marked.js.url=https://cdn.jsdelivr.net/npm/marked@{version}/lib/marked.umd.js
marked-highlight.url=https://cdn.jsdelivr.net/npm/marked-highlight@{version}/lib/index.umd.js
highlight.js.url=https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/highlight.min.js
highlight.js.css.url=https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/styles/github.min.css

# Last updated
last_updated=%s
`,
		versions["marked"],
		versions["marked-highlight"],
		versions["highlight.js"],
		time.Now().Format("2006-01-02"))
	return os.WriteFile(path, []byte(content), 0644)
}

func main() {
	fmt.Println("🔄 Starting vendor update...")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		fmt.Println("❌ Error creating vendor dir:", err)
		os.Exit(1)
	}
	versions := make(map[string]string)
	for _, pkg := range npmPackages {
		fmt.Printf("🔍 Fetching latest version for %s... ", pkg.name)
		ver, err := getLatestVersion(pkg.name)
		if err != nil {
			fmt.Printf("❌\n   Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ %s\n", ver)
		versions[pkg.name] = ver
		if pkg.cdnJs != "" {
			var jsOut string
			if pkg.name == "highlight.js" {
				jsOut = filepath.Join(vendorDir, "highlight.min.js")
			} else {
				jsOut = filepath.Join(vendorDir, pkg.name+".min.js")
			}
			jsUrl := fmt.Sprintf(pkg.cdnJs, ver)
			fmt.Printf("📥 Downloading %s to %s...\n", jsUrl, jsOut)
			if err := downloadFile(jsUrl, jsOut, pkg.minSizeJs); err != nil {
				fmt.Println("❌ Error:", err)
				os.Exit(1)
			}
			fmt.Printf("✅ %s downloaded\n", jsOut)
		}
		if pkg.cdnCss != "" {
			cssUrl := fmt.Sprintf(pkg.cdnCss, ver)
			cssOut := filepath.Join(vendorDir, "github.min.css")
			fmt.Printf("📥 Downloading %s to %s...\n", cssUrl, cssOut)
			if err := downloadFile(cssUrl, cssOut, pkg.minSizeCss); err != nil {
				fmt.Println("❌ Error:", err)
				os.Exit(1)
			}
			fmt.Printf("✅ %s downloaded\n", cssOut)
		}
	}
	fmt.Println("📝 Writing versions.txt...")
	if err := writeVersionsTxt(versions); err != nil {
		fmt.Println("❌ Error writing versions.txt:", err)
		os.Exit(1)
	}
	fmt.Println("🎉 Vendor files updated successfully!")
}
