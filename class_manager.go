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
	classes         map[string]*JadwalConfig // key: ID kelas kanonikal (contoh: "D4-TI-SMT3-A")
	aliases         map[string]string        // key: variasi alias (contoh: "3A", "D4-TI-3A", "SMT 3 A"), value: ID kanonikal
	defaultClass    string
	overrideManager *OverrideManager
}

// NormalizeClassID membersihkan dan menstandarkan ID kelas (uppercase, trim spasi, hilangkan formatting WhatsApp/markdown)
func NormalizeClassID(raw string) string {
	s := strings.TrimSpace(raw)
	// Hilangkan karakter pembungkus formatting WhatsApp/markdown yang sering ter-copy paste (`, [, ], *, _, ~, ", ', dll.)
	s = strings.Trim(s, "`[]*~_\"'() ")
	s = strings.ToUpper(s)
	if strings.HasPrefix(s, "KELAS ") {
		s = strings.TrimPrefix(s, "KELAS ")
	} else if strings.HasPrefix(s, "KELAS-") {
		s = strings.TrimPrefix(s, "KELAS-")
	}
	s = strings.Trim(s, "`[]*~_\"'() ")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}

// NewClassManager memindai direktori dirPath untuk memuat seluruh file jadwal kelas (*.json).
// Jika direktori kosong atau tidak ditemukan, sistem akan mencoba memuat fallbackSingleFile.
func NewClassManager(dirPath string, fallbackSingleFile string) (*ClassManager, error) {
	cm := &ClassManager{
		dirPath: dirPath,
		classes: make(map[string]*JadwalConfig),
		aliases: make(map[string]string),
	}

	loadedCount, err := cm.loadFromDirectory(dirPath)
	if err != nil || loadedCount == 0 {
		// Jika direktori tidak ada atau tidak ada file JSON, coba gunakan fallback file tunggal
		if fallbackSingleFile != "" {
			if _, statErr := os.Stat(fallbackSingleFile); statErr == nil {
				cfg, loadErr := LoadJadwal(fallbackSingleFile)
				if loadErr == nil {
					cm.classes["D4-TI-SMT3-A"] = cfg
					cm.classes["3A"] = cfg
					cm.classes["DEFAULT"] = cfg
					cm.aliases["3A"] = "D4-TI-SMT3-A"
					cm.aliases["D4-TI-3A"] = "D4-TI-SMT3-A"
					cm.aliases["DEFAULT"] = "D4-TI-SMT3-A"
					cm.defaultClass = "D4-TI-SMT3-A"
					return cm, nil
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca direktori jadwal '%s': %w", dirPath, err)
		}
		return nil, fmt.Errorf("tidak ditemukan file konfigurasi jadwal (*.json) di direktori '%s'", dirPath)
	}

	// Tentukan defaultClass: utamakan "D4-TI-SMT3-A" atau "3A", jika tidak ada gunakan kelas pertama secara alfabetis
	if _, ok := cm.classes["D4-TI-SMT3-A"]; ok {
		cm.defaultClass = "D4-TI-SMT3-A"
	} else if _, ok := cm.classes["3A"]; ok {
		cm.defaultClass = "3A"
	} else {
		keys := cm.ListClasses()
		if len(keys) > 0 {
			cm.defaultClass = keys[0]
		}
	}

	return cm, nil
}

// loadFromDirectory membaca semua file .json di folder target dan mendaftarkan kelas & alias
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
		cm.registerAliases(classID)
		count++
	}

	return count, nil
}

// registerAliases mendaftarkan berbagai pola penulisan singkat dan alias umum untuk satu ID kelas kanonikal
func (cm *ClassManager) registerAliases(canonicalID string) {
	norm := NormalizeClassID(canonicalID)
	// Bentuk standar: e.g. D4-TI-SMT3-A atau D3-TI-SMT1-B
	parts := strings.Split(norm, "-")
	if len(parts) >= 3 {
		prodi := parts[0] // "D4" atau "D3"
		subProdi := ""
		smtPart := ""
		kelasPart := parts[len(parts)-1] // "A", "B", "C", "D"

		if len(parts) == 4 {
			subProdi = parts[1] // "TI"
			smtPart = parts[2]  // "SMT3"
		} else if len(parts) == 3 {
			smtPart = parts[1] // "SMT3"
		}

		semester := strings.TrimPrefix(smtPart, "SMT")
		semester = strings.TrimPrefix(semester, "SEM")

		// 1. Variasi dengan dan tanpa "TI":
		if subProdi != "" {
			cm.aliases[fmt.Sprintf("%s-%s-%s%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s-%s-%s-%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s-%s-SEM%s-%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s-%s-SEM%s%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s-%s-SMT%s%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s%sSMT%s%s", prodi, subProdi, semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s%s%s%s", prodi, subProdi, semester, kelasPart)] = norm
		}

		cm.aliases[fmt.Sprintf("%s-%s%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s-%s-%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s-SMT%s-%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s-SEM%s-%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s-SMT%s%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s-SEM%s%s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s%s%s", prodi, semester, kelasPart)] = norm

		// 2. Variasi berbasis spasi:
		cm.aliases[fmt.Sprintf("%s SMT %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s SEM %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s %s%s", prodi, semester, kelasPart)] = norm

		// 3. Khusus Prodi D4 (prodi utama/default), daftarkan alias langsung tanpa prefix prodi:
		if prodi == "D4" {
			cm.aliases[fmt.Sprintf("SMT%s%s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEM%s%s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SMT%s-%s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEM%s-%s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SMT %s %s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEM %s %s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEMESTER %s %s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEMESTER %s%s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("SEMESTER %s KELAS %s", semester, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s%s", semester, kelasPart)] = norm       // e.g. "3A", "3B"
			cm.aliases[fmt.Sprintf("%s %s", semester, kelasPart)] = norm      // e.g. "3 A", "3 B"
			cm.aliases[fmt.Sprintf("KELAS %s%s", semester, kelasPart)] = norm // e.g. "KELAS 3A"
		}

		// 4. Khusus SMT3: Daftarkan alias kelas tingkat 2 (misal: "2A", "2B", "KELAS 2A")
		// agar mahasiswa yang menyebut Tingkat 2 / Kelas 2 otomatis terhubung ke SMT3 tanpa menimpa alias kelas lain.
		if semester == "3" {
			if subProdi != "" {
				cm.aliases[fmt.Sprintf("%s-%s-2%s", prodi, subProdi, kelasPart)] = norm
			}
			cm.aliases[fmt.Sprintf("%s-2%s", prodi, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s 2 %s", prodi, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s 2%s", prodi, kelasPart)] = norm

			if prodi == "D4" {
				cm.aliases[fmt.Sprintf("2%s", kelasPart)] = norm                // e.g. "2A"
				cm.aliases[fmt.Sprintf("2 %s", kelasPart)] = norm               // e.g. "2 A"
				cm.aliases[fmt.Sprintf("KELAS 2%s", kelasPart)] = norm          // e.g. "KELAS 2A"
				cm.aliases[fmt.Sprintf("TINGKAT 2 %s", kelasPart)] = norm       // e.g. "TINGKAT 2 A"
				cm.aliases[fmt.Sprintf("TINGKAT 2%s", kelasPart)] = norm        // e.g. "TINGKAT 2A"
				cm.aliases[fmt.Sprintf("TINGKAT 2 KELAS %s", kelasPart)] = norm // e.g. "TINGKAT 2 KELAS A"
			}
		}
	}
}

// ResolveClassID memetakan input pengguna (apapun variasinya) ke ID kelas kanonikal (contoh: "3a" -> "D4-TI-SMT3-A")
func (cm *ClassManager) ResolveClassID(input string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.resolveInternal(input)
}

func (cm *ClassManager) resolveInternal(input string) string {
	raw := strings.TrimSpace(strings.ToUpper(input))
	if raw == "" {
		return ""
	}

	norm := NormalizeClassID(raw)

	// 1. Cek langsung ke map kelas kanonikal
	if _, ok := cm.classes[norm]; ok {
		return norm
	}

	// 2. Cek langsung ke map alias
	if canonical, ok := cm.aliases[norm]; ok {
		if _, exists := cm.classes[canonical]; exists {
			return canonical
		}
	}

	// 3. Normalisasi karakter separator & kata kunci
	clean := strings.ReplaceAll(raw, "_", " ")
	clean = strings.ReplaceAll(clean, "/", " ")
	clean = strings.ReplaceAll(clean, ".", " ")
	clean = strings.ReplaceAll(clean, "-", " ")
	clean = strings.ReplaceAll(clean, "KELAS", " ")
	clean = strings.ReplaceAll(clean, "SEMESTER", "SMT")
	clean = strings.ReplaceAll(clean, "SEM", "SMT")
	fields := strings.Fields(clean)

	joinedCompact := strings.Join(fields, "")
	if canonical, ok := cm.aliases[joinedCompact]; ok {
		return canonical
	}

	joinedHyphen := strings.Join(fields, "-")
	if canonical, ok := cm.aliases[joinedHyphen]; ok {
		return canonical
	}

	joinedSpace := strings.Join(fields, " ")
	if canonical, ok := cm.aliases[joinedSpace]; ok {
		return canonical
	}

	// 4. Deteksi komponen: Prodi (D3/D4), Semester (1/3/5/7), dan Huruf Kelas (A/B/C/D)
	isD3 := strings.Contains(clean, "D3")
	prodiPrefix := "D4-TI-"
	if isD3 && !strings.Contains(clean, "D4") {
		prodiPrefix = "D3-TI-"
	}

	var semDigit string
	for _, f := range fields {
		fTrimmed := strings.TrimPrefix(f, "SMT")
		if len(fTrimmed) >= 1 && (fTrimmed[0] >= '1' && fTrimmed[0] <= '8') {
			semDigit = string(fTrimmed[0])
			break
		}
	}

	var classLetter string
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if len(f) == 1 && f[0] >= 'A' && f[0] <= 'D' {
			classLetter = f
			break
		}
		if len(f) == 2 && (f[0] >= '1' && f[0] <= '8') && (f[1] >= 'A' && f[1] <= 'D') {
			semDigit = string(f[0])
			classLetter = string(f[1])
			break
		}
		if strings.HasPrefix(f, "SMT") && len(f) >= 5 {
			s := strings.TrimPrefix(f, "SMT")
			if len(s) == 2 && (s[0] >= '1' && s[0] <= '8') && (s[1] >= 'A' && s[1] <= 'D') {
				semDigit = string(s[0])
				classLetter = string(s[1])
				break
			}
		}
	}

	if semDigit != "" && classLetter != "" {
		targetCanonical := fmt.Sprintf("%sSMT%s-%s", prodiPrefix, semDigit, classLetter)
		if _, ok := cm.classes[targetCanonical]; ok {
			return targetCanonical
		}
	}

	return ""
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

// GetClass mengambil jadwal kelas berdasarkan ID kelas (kanonikal atau alias case-insensitive)
func (cm *ClassManager) GetClass(classID string) (*JadwalConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	norm := NormalizeClassID(classID)
	if cfg, ok := cm.classes[norm]; ok {
		return cfg, true
	}

	canonical := cm.resolveInternal(classID)
	if canonical != "" {
		if cfg, ok := cm.classes[canonical]; ok {
			return cfg, true
		}
	}

	return nil, false
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
	for _, cfg := range cm.classes {
		return cfg
	}
	return nil
}

// GetDefaultClassID mengembalikan nama ID kelas default (contoh: "D4-TI-SMT3-A")
func (cm *ClassManager) GetDefaultClassID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.defaultClass
}

// HasClass mengecek apakah suatu kelas terdaftar di sistem (mendukung alias)
func (cm *ClassManager) HasClass(classID string) bool {
	_, ok := cm.GetClass(classID)
	return ok
}

// ListClasses mengembalikan daftar seluruh ID kelas kanonikal yang tersedia secara terurut alfabetis
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
