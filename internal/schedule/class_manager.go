package schedule

import (
	"bot-jadwal/internal/util"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

	resolvedDir := util.FindDataDir(dirPath)
	loadedCount, err := cm.loadFromDirectory(resolvedDir)
	if err != nil || loadedCount == 0 {
		// Jika direktori tidak ada atau tidak ada file JSON, coba gunakan fallback file tunggal
		if fallbackSingleFile != "" {
			resolvedFallback := util.FindDataDir(fallbackSingleFile)
			if _, statErr := os.Stat(resolvedFallback); statErr == nil {
				cfg, loadErr := LoadJadwal(resolvedFallback)
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
	parts := strings.Split(norm, "-")
	if len(parts) >= 3 {
		prodi := parts[0]
		subProdi := ""
		smtPart := ""
		kelasPart := parts[len(parts)-1]

		if len(parts) == 4 {
			subProdi = parts[1]
			smtPart = parts[2]
		} else if len(parts) == 3 {
			smtPart = parts[1]
		}

		semester := strings.TrimPrefix(smtPart, "SMT")
		semester = strings.TrimPrefix(semester, "SEM")

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

		cm.aliases[fmt.Sprintf("%s SMT %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s SEM %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s %s %s", prodi, semester, kelasPart)] = norm
		cm.aliases[fmt.Sprintf("%s %s%s", prodi, semester, kelasPart)] = norm

		// Alias tingkat 2 dipetakan ke semester 3 jika prodi dicantumkan (contoh: D4 2A -> D4-TI-SMT3-A)
		if semester == "3" {
			if subProdi != "" {
				cm.aliases[fmt.Sprintf("%s-%s-2%s", prodi, subProdi, kelasPart)] = norm
			}
			cm.aliases[fmt.Sprintf("%s-2%s", prodi, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s 2 %s", prodi, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s 2%s", prodi, kelasPart)] = norm
		}

		// Alias angkatan / cohort permanen (misal: D4-TI-2025-A -> D4-TI-SMT3-A)
		if semInt, err := strconv.Atoi(semester); err == nil && semInt > 0 {
			academicStartYear := time.Now().Year()
			if time.Now().Month() < time.September {
				academicStartYear--
			}
			cohortYear := academicStartYear - ((semInt - 1) / 2)
			if subProdi != "" {
				cm.aliases[fmt.Sprintf("%s-%s-%d-%s", prodi, subProdi, cohortYear, kelasPart)] = norm
				cm.aliases[fmt.Sprintf("%s-%s-%d%s", prodi, subProdi, cohortYear, kelasPart)] = norm
				cm.aliases[fmt.Sprintf("%s%s%d%s", prodi, subProdi, cohortYear, kelasPart)] = norm
				cm.aliases[fmt.Sprintf("%s %s %d %s", prodi, subProdi, cohortYear, kelasPart)] = norm
			}
			cm.aliases[fmt.Sprintf("%s-%d-%s", prodi, cohortYear, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s-%d%s", prodi, cohortYear, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s %d %s", prodi, cohortYear, kelasPart)] = norm
			cm.aliases[fmt.Sprintf("%s %d%s", prodi, cohortYear, kelasPart)] = norm
		}
	}
}

// ResolveClassCandidates mencari seluruh kandidat ID kelas kanonikal yang cocok dengan input pengguna.
// Jika input spesifik (misal "D4 3A", "D3 3A", atau "D4-TI-SMT3-A"), mengembalikan 1 kandidat.
// Jika input ambigu (misal "smt 3 a" atau "3a" yang tersedia di D4 dan D3), mengembalikan seluruh kandidat yang cocok.
// Jika tidak cocok sama sekali, mengembalikan slice kosong.
func (cm *ClassManager) ResolveClassCandidates(input string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.resolveCandidatesInternal(input)
}

// ResolveClassID memetakan input pengguna ke ID kelas kanonikal (contoh: "d4 3a" -> "D4-TI-SMT3-A").
// Jika input memiliki beberapa kandidat (ambigu seperti "3a"), mengembalikan kandidat pertama (default D4) untuk kompatibilitas pencarian.
// Untuk validasi interaktif yang mendeteksi ambiguitas (seperti pada perintah !setkelas), gunakan ResolveClassCandidates.
func (cm *ClassManager) ResolveClassID(input string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.resolveInternal(input)
}

func (cm *ClassManager) resolveInternal(input string) string {
	candidates := cm.resolveCandidatesInternal(input)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func (cm *ClassManager) resolveCandidatesInternal(input string) []string {
	raw := strings.TrimSpace(strings.ToUpper(input))
	if raw == "" {
		return nil
	}

	norm := NormalizeClassID(raw)

	if _, ok := cm.classes[norm]; ok {
		return []string{norm}
	}

	if canonical, ok := cm.aliases[norm]; ok {
		if _, exists := cm.classes[canonical]; exists {
			return []string{canonical}
		}
	}

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
		if _, exists := cm.classes[canonical]; exists {
			return []string{canonical}
		}
	}

	joinedHyphen := strings.Join(fields, "-")
	if canonical, ok := cm.aliases[joinedHyphen]; ok {
		if _, exists := cm.classes[canonical]; exists {
			return []string{canonical}
		}
	}

	joinedSpace := strings.Join(fields, " ")
	if canonical, ok := cm.aliases[joinedSpace]; ok {
		if _, exists := cm.classes[canonical]; exists {
			return []string{canonical}
		}
	}

	hasD3 := false
	hasD4 := false
	for _, f := range fields {
		if f == "D3" || strings.HasPrefix(f, "D3") {
			hasD3 = true
		}
		if f == "D4" || strings.HasPrefix(f, "D4") {
			hasD4 = true
		}
	}
	if !hasD3 && strings.Contains(clean, "D3") {
		hasD3 = true
	}
	if !hasD4 && strings.Contains(clean, "D4") {
		hasD4 = true
	}

	var semDigit string
	for _, f := range fields {
		fTrimmed := strings.TrimPrefix(f, "SMT")
		if len(fTrimmed) == 4 {
			if yr, err := strconv.Atoi(fTrimmed); err == nil && yr >= 2000 && yr <= 2099 {
				academicStartYear := time.Now().Year()
				if time.Now().Month() < time.September {
					academicStartYear--
				}
				s := (academicStartYear-yr)*2 + 1
				if s >= 1 && s <= 8 {
					semDigit = strconv.Itoa(s)
					break
				}
			}
		}
		if len(fTrimmed) >= 1 && (fTrimmed[0] >= '1' && fTrimmed[0] <= '8') {
			semDigit = string(fTrimmed[0])
			break
		}
	}

	if semDigit == "" && (strings.Contains(clean, "TINGKAT 2") || strings.Contains(clean, "TINGKAT2")) {
		semDigit = "3"
	}

	var classLetter string
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if len(f) == 1 && f[0] >= 'A' && f[0] <= 'D' {
			classLetter = f
			break
		}
		if len(f) == 2 && (f[0] >= '1' && f[0] <= '8') && (f[1] >= 'A' && f[1] <= 'D') {
			if semDigit == "" {
				semDigit = string(f[0])
			}
			classLetter = string(f[1])
			break
		}
		if strings.HasPrefix(f, "SMT") && len(f) >= 5 {
			s := strings.TrimPrefix(f, "SMT")
			if len(s) == 2 && (s[0] >= '1' && s[0] <= '8') && (s[1] >= 'A' && s[1] <= 'D') {
				if semDigit == "" {
					semDigit = string(s[0])
				}
				classLetter = string(s[1])
				break
			}
		}
	}

	if semDigit == "" && len(fields) > 0 {
		for _, f := range fields {
			if len(f) == 2 && (f[0] >= '1' && f[0] <= '8') && (f[1] >= 'A' && f[1] <= 'D') {
				semDigit = string(f[0])
				classLetter = string(f[1])
				break
			}
		}
	}

	// Jika pengguna memasukkan "2A" atau semester 2, tapi di sistem aktif semester ganjil (SMT3):
	// petakan semester 2 / tingkat 2 ke semester 3 jika SMT2 tidak ada di daftar kelas.
	if semDigit == "2" {
		hasSMT2 := false
		for k := range cm.classes {
			if strings.Contains(k, "SMT2-") {
				hasSMT2 = true
				break
			}
		}
		if !hasSMT2 {
			semDigit = "3"
		}
	}

	if semDigit != "" && classLetter != "" {
		var prodisToCheck []string
		if hasD3 && !hasD4 {
			prodisToCheck = []string{"D3"}
		} else if hasD4 && !hasD3 {
			prodisToCheck = []string{"D4"}
		} else {
			prodisToCheck = []string{"D4", "D3"}
		}

		var candidates []string
		for _, p := range prodisToCheck {
			targetCanonical := fmt.Sprintf("%s-TI-SMT%s-%s", p, semDigit, classLetter)
			if _, ok := cm.classes[targetCanonical]; ok {
				candidates = append(candidates, targetCanonical)
			}
		}
		return candidates
	}

	return nil
}

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

func (cm *ClassManager) GetDefaultClassID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.defaultClass
}

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
