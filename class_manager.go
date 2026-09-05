package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ClassManager mengelola banyak jadwal kelas (multi-tenant) yang dimuat dari file JSON
type ClassManager struct {
	mu              sync.RWMutex
	dirPath         string
	classes         map[string]*JadwalConfig // key: ID kelas huruf kapital (contoh: "3A", "3B")
	defaultClass    string
	overrideManager *OverrideManager
}

// NormalizeClassID membersihkan dan menstandarkan ID kelas (uppercase, trim spasi, hilangkan prefix 'kelas')
func NormalizeClassID(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ToUpper(s)
	if strings.HasPrefix(s, "KELAS ") {
		s = strings.TrimPrefix(s, "KELAS ")
	} else if strings.HasPrefix(s, "KELAS-") {
		s = strings.TrimPrefix(s, "KELAS-")
	}
	return strings.TrimSpace(s)
}

// NewClassManager memindai direktori dirPath untuk memuat seluruh file jadwal kelas (*.json).
// Jika direktori kosong atau tidak ditemukan, sistem akan mencoba memuat fallbackSingleFile.
func NewClassManager(dirPath string, fallbackSingleFile string) (*ClassManager, error) {
	cm := &ClassManager{
		dirPath: dirPath,
		classes: make(map[string]*JadwalConfig),
	}

	loadedCount, err := cm.loadFromDirectory(dirPath)
	if err != nil || loadedCount == 0 {
		// Jika direktori tidak ada atau tidak ada file JSON, coba gunakan fallback file tunggal
		if fallbackSingleFile != "" {
			if _, statErr := os.Stat(fallbackSingleFile); statErr == nil {
				cfg, loadErr := LoadJadwal(fallbackSingleFile)
				if loadErr == nil {
					cm.classes["3A"] = cfg
					cm.classes["DEFAULT"] = cfg
					cm.defaultClass = "3A"
					return cm, nil
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca direktori jadwal '%s': %w", dirPath, err)
		}
		return nil, fmt.Errorf("tidak ditemukan file konfigurasi jadwal (*.json) di direktori '%s'", dirPath)
	}

	// Tentukan defaultClass: utamakan "3A", jika tidak ada gunakan kelas pertama secara alfabetis
	if _, ok := cm.classes["3A"]; ok {
		cm.defaultClass = "3A"
	} else {
		keys := cm.ListClasses()
		if len(keys) > 0 {
			cm.defaultClass = keys[0]
		}
	}

	return cm, nil
}

// loadFromDirectory membaca semua file .json di folder target
func (cm *ClassManager) loadFromDirectory(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}

		classID := NormalizeClassID(strings.TrimSuffix(name, filepath.Ext(name)))
		if classID == "" {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		cfg, err := LoadJadwal(fullPath)
		if err != nil {
			fmt.Printf("⚠️ [ClassManager] Gagal memuat jadwal kelas '%s' (%s): %v\n", classID, fullPath, err)
			continue
		}

		cm.classes[classID] = cfg
		count++
	}

	return count, nil
}

// SetOverrideManager menghubungkan OverrideManager ke seluruh jadwal kelas yang telah dimuat
func (cm *ClassManager) SetOverrideManager(om *OverrideManager) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.overrideManager = om
	for _, cfg := range cm.classes {
		cfg.SetOverrideManager(om)
	}
}

// GetClass mengambil jadwal kelas berdasarkan ID kelas (case-insensitive)
func (cm *ClassManager) GetClass(classID string) (*JadwalConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	norm := NormalizeClassID(classID)
	cfg, ok := cm.classes[norm]
	return cfg, ok
}

// GetClassOrDefault mengambil jadwal kelas berdasarkan ID kelas, atau kelas default jika tidak ditemukan
func (cm *ClassManager) GetClassOrDefault(classID string) *JadwalConfig {
	if classID != "" {
		if cfg, ok := cm.GetClass(classID); ok {
			return cfg
		}
	}
	return cm.GetDefaultClass()
}

// GetDefaultClass mengambil jadwal kelas default
func (cm *ClassManager) GetDefaultClass() *JadwalConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cfg, ok := cm.classes[cm.defaultClass]; ok {
		return cfg
	}
	// Fallback ke kelas apa saja yang tersedia
	for _, cfg := range cm.classes {
		return cfg
	}
	return nil
}

// GetDefaultClassID mengembalikan nama ID kelas default (contoh: "3A")
func (cm *ClassManager) GetDefaultClassID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.defaultClass
}

// HasClass mengecek apakah suatu kelas terdaftar di sistem
func (cm *ClassManager) HasClass(classID string) bool {
	_, ok := cm.GetClass(classID)
	return ok
}

// ListClasses mengembalikan daftar seluruh ID kelas yang tersedia secara terurut alfabetis
func (cm *ClassManager) ListClasses() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	list := make([]string, 0, len(cm.classes))
	for id := range cm.classes {
		if id == "DEFAULT" {
			continue
		}
		list = append(list, id)
	}
	sort.Strings(list)
	return list
}

// ReloadAll membaca ulang seluruh file JSON dari disk tanpa mematikan bot
func (cm *ClassManager) ReloadAll() (int, []error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errs []error
	count := 0

	for id, cfg := range cm.classes {
		if cfg.FilePath == "" {
			continue
		}
		_, err := cfg.Reload()
		if err != nil {
			errs = append(errs, fmt.Errorf("kelas %s: %w", id, err))
		} else {
			if cm.overrideManager != nil {
				cfg.SetOverrideManager(cm.overrideManager)
			}
			count++
		}
	}

	return count, errs
}
