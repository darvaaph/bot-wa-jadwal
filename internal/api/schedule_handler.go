package api

import (
	"bot-jadwal/internal/util"
	"net/http"
	"strings"
	"time"
)

// ClassesResponse adalah format balasan untuk endpoint GET /api/classes
type ClassesResponse struct {
	Status string              `json:"status"`
	Data   ClassesDataResponse `json:"data"`
}

// ClassesDataResponse adalah payload data kelas yang terdaftar
type ClassesDataResponse struct {
	DefaultClass string   `json:"default_class"`
	TotalClasses int      `json:"total_classes"`
	Classes      []string `json:"classes"`
}

// ScheduleItemResponse merepresentasikan entri jadwal perkuliahan individual untuk respons API
type ScheduleItemResponse struct {
	Hari   string `json:"hari"`
	Jam    string `json:"jam"`
	Matkul string `json:"matkul"`
	Dosen  string `json:"dosen"`
	Ruang  string `json:"ruang"`
}

// ScheduleResponse adalah format balasan untuk endpoint GET /api/schedule
type ScheduleResponse struct {
	Status string                 `json:"status"`
	Class  string                 `json:"class"`
	Day    string                 `json:"day"`
	Data   []ScheduleItemResponse `json:"data"`
}

// handleClasses menyajikan daftar seluruh kode kelas kanonikal beserta kelas default
func (s *Server) handleClasses(w http.ResponseWriter, r *http.Request) {
	defaultClass := ""
	classes := make([]string, 0)

	if s.classManager != nil {
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

// handleSchedule menyajikan jadwal perkuliahan berdasarkan kelas dan filter hari
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
