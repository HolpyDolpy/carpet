//go:build windows

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func shellOpen(target string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	path, _ := syscall.UTF16PtrFromString(target)
	v, _, _ := proc.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(path)), 0, 0, 1)
	if v <= 32 {
		return fmt.Errorf("Windows cannot open item (code %d)", v)
	}
	return nil
}
func openBrowser(url string) error { return shellOpen(url) }
func openFolder(path string) error { return shellOpen(path) }
func showError(title, message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	msg, _ := syscall.UTF16PtrFromString(message)
	proc.Call(0, uintptr(unsafe.Pointer(msg)), uintptr(unsafe.Pointer(t)), 0x10)
}
func nativeFileDialog(mode string) (string, error) {
	// Native Windows Save/Open dialog. PowerShell is built into Windows 10/11; no extra runtime.
	script := `Add-Type -AssemblyName System.Windows.Forms
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$dlg = New-Object System.Windows.Forms.`
	if mode == "save" {
		script += `SaveFileDialog
$dlg.Title = 'Save carpet sales workbook / حفظ ملف المبيعات'
$dlg.FileName = 'Carpets_Ameer_Sales.xlsx'
$dlg.OverwritePrompt = $false
`
	} else {
		script += `OpenFileDialog
$dlg.Title = 'Open carpet sales workbook / فتح ملف المبيعات'
$dlg.CheckFileExists = $true
`
	}
	script += `$dlg.Filter = 'Excel Workbook (*.xlsx)|*.xlsx'
$dlg.DefaultExt = 'xlsx'
$dlg.AddExtension = $true
$dlg.InitialDirectory = [Environment]::GetFolderPath('MyDocuments')
if ($dlg.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $dlg.FileName }
`
	encoded := base64.StdEncoding.EncodeToString(utf16LEBytes(script))
	c := exec.Command("powershell.exe", "-NoProfile", "-STA", "-EncodedCommand", encoded)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Windows file picker failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\r", "")), nil
}
func utf16LEBytes(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, v := range u {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return b
}
func replaceFileAtomic(tmp, target string) error {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel.NewProc("MoveFileExW")
	s, _ := syscall.UTF16PtrFromString(tmp)
	d, _ := syscall.UTF16PtrFromString(target)
	res, _, err := proc.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(d)), uintptr(1|8)) // MOVEFILE_REPLACE_EXISTING|WRITE_THROUGH
	if res == 0 {
		return fmt.Errorf("Windows could not replace Excel file safely (close it in Excel first): %v", err)
	}
	return nil
}
func applicationUserConfigDir() string { a, _ := os.UserConfigDir(); return a }
