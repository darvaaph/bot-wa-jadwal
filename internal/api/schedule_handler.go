package api

import (
	"bot-jadwal/internal/util"
	"net/http"
	"strings"
	"time"
)

type ClassesResponse struct {
	Status string              `json:"status"`
	Data   ClassesDataResponse `json:"data"`
}

type ClassesDataResponse struct {
	DefaultClass string   `json:"default_class"`
	TotalClasses int      `json:"total_classes"`
	Classes      []string `json:"classes"`
}

type ScheduleItemResponse struct {
	Hari   string `json:"hari"`
	Jam    string `json:"jam"`
	Matkul string `json:"matkul"`
	Dosen  string `json:"dosen"`
	Ruang  string `json:"ruang"`
}

type ScheduleResponse struct {
	Status string                 `json:"status"`
	Class  string                 `json:"class"`
	Day    string                 `json:"day"`
	Data   []ScheduleItemResponse `json:"data"`
}

func (s *Server) handleClasses(w http.ResponseWriter, r *http.Request) {
	defaultClass := ""
	classes := make([]string, 0)

	if s.academicRepo != nil {
		if dbClasses, err := s.academicRepo.GetClasses(r.Context()); err == nil && len(dbClasses) > 0 {
			for _, c := range dbClasses {
				classes = append(classes, c.Code)
			}
			defaultClass = dbClasses[0].Code
		}
	}

	// Jadwal berbasis berkas tetap tersedia saat data akademik belum disemai.
	if len(classes) == 0 && s.classManager != nil {
		classes = s.classManager.ListClasses()
		if classes == nil {
			classes = make([]string, 0)
		}
		defaultClass = s.classManager.GetDefaultClassID()
	}

	resp := ClassesResponse{
		Status: "success",
		Data: ClassesDataResponse{
			DefaultClass: defaultClass,
			TotalClasses: len(classes),
			Classes:      classes,
		},
	}

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if s.classManager == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"status":  "error",
			"message": "Kelas tidak ditemukan",
		})
		return
	}

	classQuery := strings.TrimSpace(r.URL.Query().Get("class"))
	if classQuery == "" {
		classQuery = s.classManager.GetDefaultClassID()
	}

	cfg, ok := s.classManager.GetClass(classQuery)
	if !ok || cfg == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"status":  "error",
			"message": "Kelas tidak ditemukan",
		})
		return
	}

	canonicalClassID := s.classManager.ResolveClassID(classQuery)
	if canonicalClassID == "" {
		canonicalClassID = classQuery
	}

	dayQuery := strings.TrimSpace(r.URL.Query().Get("day"))
	var targetDay string
	var respDay string

	if strings.EqualFold(dayQuery, "today") || strings.EqualFold(dayQuery, "hari ini") {
		targetDay = util.GetHariIndonesia(time.Now())
		respDay = targetDay
	} else if dayQuery != "" && !strings.EqualFold(dayQuery, "all") && !strings.EqualFold(dayQuery, "semua") {
		targetDay = normalizeDayInput(dayQuery)
		respDay = targetDay
	} else {
		respDay = "all"
	}

	items := make([]ScheduleItemResponse, 0)
	for _, item := range cfg.Jadwal {
		if targetDay != "" && !strings.EqualFold(item.Hari, targetDay) {
			continue
		}

		matkul := item.NamaMatkul
		if matkul == "" {
			if m, exists := cfg.MataKuliah[item.KodeMatkul]; exists && m != "" {
				matkul = m
			} else {
				matkul = item.KodeMatkul
			}
		}

		dosen := item.Dosen
		if dosen == "" {
			if d, exists := cfg.Dosen[item.InisialDosen]; exists && d != "" {
				dosen = d
			} else {
				dosen = item.InisialDosen
			}
		}

		items = append(items, ScheduleItemResponse{
			Hari:   item.Hari,
			Jam:    item.Jam,
			Matkul: matkul,
			Dosen:  dosen,
			Ruang:  item.Ruang,
		})
	}

	resp := ScheduleResponse{
		Status: "success",
		Class:  canonicalClassID,
		Day:    respDay,
		Data:   items,
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// normalizeDayInput menstandarkan input nama hari ke format huruf kapital awal bahasa Indonesia
func normalizeDayInput(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "senin", "monday":
		return "Senin"
	case "selasa", "tuesday":
		return "Selasa"
	case "rabu", "wednesday":
		return "Rabu"
	case "kamis", "thursday":
		return "Kamis"
	case "jumat", "jum'at", "friday":
		return "Jumat"
	case "sabtu", "saturday":
		return "Sabtu"
	case "minggu", "sunday":
		return "Minggu"
	default:
		return input
	}
}
