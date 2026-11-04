//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func openBrowser(url string) error    { return exec.Command("xdg-open", url).Start() }
func openFolder(path string) error    { return exec.Command("xdg-open", path).Start() }
func showError(title, message string) { fmt.Fprintln(os.Stderr, title+": "+message) }
func nativeFileDialog(mode string) (string, error) {
	if os.Getenv("CARPETS_TEST_MODE") == "1" {
		return os.Getenv("CARPETS_TEST_PICKER_" + strings.ToUpper(mode)), nil
	}
	return "", fmt.Errorf("Native file picker is only available in the Windows build")
}
func replaceFileAtomic(tmp, target string) error { return os.Rename(tmp, target) }

func runDesktop(url, dataFolder string, onExit func()) error {
    return openBrowser(url)
}
