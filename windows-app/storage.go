package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type workbookLink struct {
	Path        string `json:"path"`
	LastHash    string `json:"hash"`
	LastSavedAt string `json:"lastSavedAt"`
}

type snapshot struct {
	Sales     []json.RawMessage `json:"sales"`
	Lang      string            `json:"lang"`
	Period    string            `json:"period"`
	Dirty     bool              `json:"dirty"`
	UpdatedAt int64             `json:"updatedAt"`
}

func jsonEncode(w io.Writer, v any) { _ = json.NewEncoder(w).Encode(v) }
func sha(data []byte) string        { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func (a *app) linkFile() string     { return filepath.Join(a.folder, "workbook-link.json") }
func (a *app) snapshotFile() string { return filepath.Join(a.folder, "sales-draft.json") }
func (a *app) loadLink() error {
	b, err := os.ReadFile(a.linkFile())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, &a.link); err != nil {
		return fmt.Errorf("Linked workbook settings are damaged: %w", err)
	}
	if a.link.Path != "" && !strings.EqualFold(filepath.Ext(a.link.Path), ".xlsx") {
		return fmt.Errorf("The linked workbook path is invalid")
	}
	return nil
}
func (a *app) writeLink(link workbookLink) error {
	raw, err := json.MarshalIndent(link, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWriteFile(a.linkFile(), raw); err != nil {
		return err
	}
	a.link = link
	return nil
}

func writeTemp(target string, data []byte) (string, error) {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(parent, ".carpets-safe-*.tmp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	cleanup := func(e error) (string, error) { _ = f.Close(); _ = os.Remove(name); return "", e }
	if _, err = f.Write(data); err != nil {
		return cleanup(err)
	}
	if err = f.Sync(); err != nil {
		return cleanup(err)
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
func atomicWriteFile(target string, data []byte) error {
	tmp, err := writeTemp(target, data)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return replaceFileAtomic(tmp, target)
}
func copyVerified(path string, data []byte) error {
	if err := atomicWriteFile(path, data); err != nil {
		return err
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if sha(disk) != sha(data) {
		return fmt.Errorf("Backup copy verification failed: %s", path)
	}
	return nil
}
func validateWorkbook(data []byte) error {
	if len(data) < 100 || len(data) > 32<<20 {
		return fmt.Errorf("Invalid Excel workbook size")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("Invalid Excel ZIP container: %w", err)
	}
	hasTypes, hasBook, hasSheet := false, false, false
	if len(z.File) > 5000 {
		return fmt.Errorf("Excel workbook has too many entries")
	}
	var size uint64
	for _, f := range z.File {
		if f.UncompressedSize64 > 80<<20 {
			return fmt.Errorf("Excel workbook contains an oversized part")
		}
		size += f.UncompressedSize64
		if size > 120<<20 {
			return fmt.Errorf("Excel workbook expands beyond safe limits")
		}
		if f.Name == "[Content_Types].xml" {
			hasTypes = true
		}
		if f.Name == "xl/workbook.xml" {
			hasBook = true
		}
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			hasSheet = true
		}
		if f.FileInfo().IsDir() {
			continue
		}
		rc, e := f.Open()
		if e != nil {
			return e
		}
		_, e = io.Copy(io.Discard, rc)
		closeErr := rc.Close()
		if e != nil {
			return fmt.Errorf("Corrupt Excel entry %s: %w", f.Name, e)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if !hasTypes || !hasBook || !hasSheet {
		return fmt.Errorf("Excel file is missing workbook components")
	}
	return nil
}
func (a *app) bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Carpets-Token") != a.token {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var draft *snapshot
	for _, draftPath := range []string{a.snapshotFile(), filepath.Join(a.folder, "sales-draft.previous.json")} {
		data, err := os.ReadFile(draftPath)
		if err == nil {
			var parsed snapshot
			if json.Unmarshal(data, &parsed) == nil && parsed.Sales != nil {
				draft = &parsed
				break
			}
		}
	}
	name := ""
	if a.link.Path != "" {
		name = filepath.Base(a.link.Path)
	}
	respond(w, 200, map[string]any{"draft": draft, "linked": a.link.Path != "", "fileName": name, "lastSavedAt": a.link.LastSavedAt, "backupFolder": filepath.Join(filepath.Dir(a.link.Path), "Carpets_Ameer_Backups")})
}
func (a *app) postSnapshot(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	raw, err := readLimited(w, r, 6<<20)
	if err != nil {
		fail(w, 400, err)
		return
	}
	var state snapshot
	if err = json.Unmarshal(raw, &state); err != nil {
		fail(w, 400, fmt.Errorf("Invalid sales draft: %w", err))
		return
	}
	if state.Sales == nil || len(state.Sales) > 20000 || state.UpdatedAt <= 0 {
		fail(w, 400, stringError("Invalid saved sales data"))
		return
	}
	if state.Lang != "ar" && state.Lang != "en" {
		fail(w, 400, stringError("Invalid interface language"))
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Keep the last known good local draft as additional recovery protection.
	if previous, readErr := os.ReadFile(a.snapshotFile()); readErr == nil && json.Valid(previous) {
		if err = atomicWriteFile(filepath.Join(a.folder, "sales-draft.previous.json"), previous); err != nil {
			fail(w, 500, fmt.Errorf("Could not preserve previous draft: %w", err))
			return
		}
	}
	if err = atomicWriteFile(a.snapshotFile(), raw); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"ok": true, "updatedAt": state.UpdatedAt})
}

func validatePath(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".xlsx") {
		return fmt.Errorf("Choose an Excel .xlsx file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if path != abs {
		return fmt.Errorf("Please choose an absolute workbook path")
	}
	return nil
}
func (a *app) selectSave(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	path, err := nativeFileDialog("save")
	if err != nil {
		fail(w, 500, err)
		return
	}
	if path == "" {
		respond(w, 200, map[string]any{"cancelled": true})
		return
	}
	if err = validatePath(path); err != nil {
		fail(w, 400, err)
		return
	}
	var previous []byte
	previous, err = os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fail(w, 500, err)
		return
	}
	// Saving over an existing workbook without importing it first could silently
	// destroy transactions (or unrelated sheets). Make Open Existing the only
	// way to connect an already-populated file.
	if err == nil {
		if validateErr := validateWorkbook(previous); validateErr != nil {
			fail(w, 400, validateErr)
			return
		}
		fail(w, 409, stringError("That Excel workbook already exists. Use 'Open existing' to import it before editing. No file was changed."))
		return
	}
	link := workbookLink{Path: path}
	if len(previous) > 0 {
		link.LastHash = sha(previous)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err = a.writeLink(link); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"linked": true, "fileName": filepath.Base(path), "exists": len(previous) > 0})
}
func (a *app) selectOpen(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	path, err := nativeFileDialog("open")
	if err != nil {
		fail(w, 500, err)
		return
	}
	if path == "" {
		w.WriteHeader(204)
		return
	}
	if err = validatePath(path); err != nil {
		fail(w, 400, err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err = validateWorkbook(data); err != nil {
		fail(w, 400, err)
		return
	}
	a.mu.Lock()
	a.pendingOpen = path
	a.mu.Unlock()
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("X-Carpets-Name", filepath.Base(path))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
func (a *app) confirmOpen(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pendingOpen == "" {
		fail(w, 400, stringError("No workbook selected"))
		return
	}
	path := a.pendingOpen
	data, err := os.ReadFile(path)
	if err != nil {
		fail(w, 500, err)
		return
	}
	if err = validateWorkbook(data); err != nil {
		fail(w, 400, err)
		return
	}
	link := workbookLink{Path: path, LastHash: sha(data)}
	if err = a.writeLink(link); err != nil {
		fail(w, 500, err)
		return
	}
	a.pendingOpen = ""
	respond(w, 200, map[string]any{"linked": true, "fileName": filepath.Base(path)})
}

func (a *app) saveWorkbook(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	data, err := readLimited(w, r, 32<<20)
	if err != nil {
		fail(w, 413, err)
		return
	}
	if err = validateWorkbook(data); err != nil {
		fail(w, 400, err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.link.Path == "" {
		fail(w, 409, stringError("Please choose an Excel file first"))
		return
	}
	backup, err := a.saveWorkbookBytes(data)
	if err != nil {
		code := 500
		if errors.Is(err, errConflict) {
			code = 409
		}
		fail(w, code, err)
		return
	}
	respond(w, 200, map[string]any{"ok": true, "fileName": filepath.Base(a.link.Path), "backup": backup, "savedAt": a.link.LastSavedAt, "sha256": a.link.LastHash})
}

var errConflict = errors.New("Excel file changed outside this app. Open that file in Carpets Ameer to import its latest data before saving again")

func (a *app) saveWorkbookBytes(data []byte) (string, error) {
	path := a.link.Path
	old, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if a.link.LastHash != "" && !exists {
		return "", fmt.Errorf("%w: the linked Excel file is missing", errConflict)
	}
	if exists && sha(old) != a.link.LastHash {
		return "", errConflict
	}
	if exists {
		if err = validateWorkbook(old); err != nil {
			return "", fmt.Errorf("Existing workbook failed integrity check: %w", err)
		}
	}
	backupDir := filepath.Join(filepath.Dir(path), "Carpets_Ameer_Backups")
	if err = os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("Cannot create backup folder: %w", err)
	}
	stamp := time.Now().Format("20060102_150405.000000000")
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	backupPath := filepath.Join(backupDir, base+"_"+stamp+"_saved.xlsx")
	// FIRST safeguard: durable, verified copy of the NEW version before touching the linked workbook.
	if err = copyVerified(backupPath, data); err != nil {
		return "", fmt.Errorf("Cannot verify new version backup; workbook left unchanged: %w", err)
	}
	var previousBackup string
	if exists {
		previousBackup = filepath.Join(backupDir, base+"_"+stamp+"_previous.xlsx")
		if err = copyVerified(previousBackup, old); err != nil {
			return "", fmt.Errorf("Cannot verify previous version backup; workbook left unchanged: %w", err)
		}
	}
	temp, err := writeTemp(path, data)
	if err != nil {
		return "", fmt.Errorf("Could not prepare workbook replacement; original unchanged: %w", err)
	}
	defer os.Remove(temp)
	pending, err := os.ReadFile(temp)
	if err != nil {
		return "", err
	}
	if sha(pending) != sha(data) {
		return "", fmt.Errorf("Temporary Excel file verification failed; original unchanged")
	}
	if err = validateWorkbook(pending); err != nil {
		return "", fmt.Errorf("Temporary workbook became invalid: %w", err)
	}
	if err = replaceFileAtomic(temp, path); err != nil {
		return "", fmt.Errorf("Could not replace workbook; original and backups preserved: %w", err)
	}
	disk, err := os.ReadFile(path)
	if err != nil || sha(disk) != sha(data) || validateWorkbook(disk) != nil {
		if exists && previousBackup != "" {
			if restoreError := atomicWriteFile(path, old); restoreError != nil {
				return "", fmt.Errorf("Workbook verification failed; restore also failed (%v). Recover from %s", restoreError, previousBackup)
			}
		}
		return "", fmt.Errorf("Workbook verification failed; backup is at %s", backupPath)
	}
	link := a.link
	link.LastHash = sha(data)
	link.LastSavedAt = time.Now().Format(time.RFC3339)
	if err = a.writeLink(link); err != nil {
		return "", fmt.Errorf("Workbook saved and verified, but could not remember the new version: %w. Backups: %s", err, backupPath)
	}
	return backupPath, nil
}
