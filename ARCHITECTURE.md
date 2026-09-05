# 🏗️ Dokumentasi Arsitektur & Panduan Pemeliharaan (Architecture & Maintainer Guide)

Dokumen ini ditujukan bagi pengembang (*developer / maintainer*) untuk memahami cara kerja internal, struktur kode, alur data, skema basis data, dan panduan menambahkan fitur baru pada Bot WhatsApp Jadwal Kuliah.

---

## 1. 🌐 Arsitektur Sistem (System Architecture)

Bot dibangun menggunakan bahasa pemrograman **Go (Golang)** dengan pendekatan arsitektur modular yang ringan (*monolithic lightweight service*) tanpa ketergantungan CGO:

```mermaid
graph TD
    WA[WhatsApp Server] <-->|Websocket E2E| WM[whatsmeow Client]
    
    subgraph "Aplikasi Bot (Go Runtime)"
        WM -->|Events| EH[Event Handler & Dispatcher<br/>(main.go)]
        
        EH -->|Jadwal & Pencarian| SC[Schedule Engine<br/>(schedule.go)]
        EH -->|Override / Pindah / Libur| OM[Override Manager<br/>(override.go)]
        EH -->|Deadline Tracker| TM[Task Manager<br/>(task.go)]
        EH -->|Konfigurasi Pengingat| RM[Reminder Manager<br/>(reminder.go)]
        
        SCHED[Background Cron 06:30 WIB<br/>(reminder.go)] -->|Broadcast| WM
        WD[Watchdog Supervisor<br/>(main.go)] -.->|Auto-Reconnect| WM
        
        SC & OM & TM & RM -.-> UT[Shared Utilities<br/>(utils.go)]
    end

    subgraph "Penyimpanan Data (Storage)"
        SC -->|Read/Reload| JSN[(jadwal.json)]
        RM -->|Read/Write| RJ[(reminder_groups.json)]
        TM -->|CRUD SQLite| DB[(tugas.db)]
        OM -->|CRUD SQLite| DB
        WM -->|Session Store| SDB[(sesi_bot.db)]
    end
```

---

## 2. 📂 Struktur & Tanggung Jawab Modul (*Directory Map*)

Seluruh logika utama berada dalam `package main` untuk menjaga kesederhanaan eksekusi dan menghindari *cyclic import*, dengan pembagian file sesuai domain tugasnya:

| File | Baris | Tanggung Jawab Utama |
| :--- | :--- | :--- |
| [main.go](file:///f:/Project/bot-jadwal/main.go) | ~400 | Inisialisasi klien WhatsApp, event listener pesan masuk, supervisor auto-reconnect (watchdog), dan *graceful shutdown*. |
| [utils.go](file:///f:/Project/bot-jadwal/utils.go) | ~200 | **Single Source of Truth** untuk fungsi pembantu: lokalisasi hari/bulan Indonesia, parser tanggal alami, ekstraksi rentang jam, dan pembacaan SQLite flexible time. |
| [schedule.go](file:///f:/Project/bot-jadwal/schedule.go) | ~1.200 | Parsing `jadwal.json`, pencarian mata kuliah/dosen/ruangan (*fuzzy*), kalkulasi kuliah aktif/berikutnya (`!next`), dan menu bot. |
| [override.go](file:///f:/Project/bot-jadwal/override.go) | ~900 | Database dan logika jadwal pengganti sementara (`!pindah`, `!kosong`, `!kuliahganti`, `!libur`), deteksi bentrok jam, dan pembatalan (`!batalganti`). |
| [task.go](file:///f:/Project/bot-jadwal/task.go) | ~900 | Database dan logika pelacak tugas SQLite (`!tugas`), filter matkul (`!tugas sbd`), perpanjangan tenggat (`!tugas edit`), badge urgensi, dan arsip riwayat selesai (`!tugas riwayat`). |
| [reminder.go](file:///f:/Project/bot-jadwal/reminder.go) | ~250 | Scheduler goroutine pengingat otomatis pagi (06:30 WIB), penyusun pesan pengingat harian, dan penyertaan peringatan tugas mendesak. |

---

## 3. 🗄️ Skema Basis Data (Database Schemas)

### A. Tabel `tasks` (Database: `tugas.db`)
Menyimpan seluruh catatan tugas kelas maupun catatan tugas pribadi mahasiswa di DM:

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_jid TEXT NOT NULL,          -- JID grup (cth: 123@g.us) atau JID user (cth: 62812@s.whatsapp.net)
    is_group BOOLEAN NOT NULL,         -- true jika di grup, false jika di DM pribadi
    matkul TEXT NOT NULL,             -- Nama mata kuliah (cth: SISTEM BASIS DATA)
    deskripsi TEXT NOT NULL,          -- Judul / instruksi tugas
    deadline TEXT NOT NULL,           -- Label teks tenggat (cth: "Jumat, 11 Sep 23:59 WIB")
    deadline_at DATETIME,             -- Timestamp UTC/Local untuk sorting dan kalkulasi countdown
    created_by TEXT NOT NULL,         -- Pengirim yang membuat tugas
    is_done BOOLEAN DEFAULT 0,        -- 0 = aktif, 1 = selesai (masuk riwayat / arsip)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tasks_scope ON tasks(scope_jid, is_done);
```

### B. Tabel `schedule_overrides` (Database: `tugas.db`)
Menyimpan perubahan jadwal dinamis yang hanya mengikat pada tanggal tertentu:

```sql
CREATE TABLE IF NOT EXISTS schedule_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_jid TEXT NOT NULL,          -- JID grup
    override_type TEXT NOT NULL,      -- RESCHEDULE, CANCEL, EXTRA, HOLIDAY
    kode_matkul TEXT NOT NULL,        -- Kode matkul (atau "LIBUR")
    nama_matkul TEXT NOT NULL,        -- Nama matkul (atau "LIBUR SEHARIAN")
    dosen TEXT NOT NULL,
    inisial_dosen TEXT NOT NULL,
    orig_date TEXT NOT NULL,          -- YYYY-MM-DD tanggal asal
    orig_jam TEXT NOT NULL,           -- Jam asal (cth: "07:00 - 08:40")
    target_date TEXT NOT NULL,        -- YYYY-MM-DD tanggal berlakunya perubahan
    new_jam TEXT NOT NULL,            -- Jam baru (cth: "13:00 - 14:40")
    ruang TEXT NOT NULL,              -- Ruangan baru
    alasan TEXT NOT NULL,             -- Alasan perpindahan / nama libur
    created_by TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_overrides_scope ON schedule_overrides(scope_jid, target_date);
```

### C. File `reminder_groups.json`
Menyimpan daftar grup WhatsApp yang berlangganan broadcast jadwal pagi:
```json
{
  "hour": 6,
  "minute": 30,
  "groups": [
    {
      "jid": "120363xxxxxx@g.us",
      "name": "D4 Teknik Informatika 3A",
      "added_at": "2026-09-04T08:00:00Z"
    }
  ]
}
```

---

## 4. 🔄 Alur Pemrosesan Pesan (*Message Processing Lifecycle*)

Saat sebuah pesan masuk dari WhatsApp:
1. **Filter Awal:** `main.go` mengabaikan pesan dari bot sendiri (`v.Info.IsFromMe`).
2. **Ekstraksi Teks:** Pesan diambil dari `Conversation` atau `ExtendedTextMessage`.
3. **Pembersihan Prefix:** Fungsi `cleanCommandPrefix(msg)` di [utils.go](file:///f:/Project/bot-jadwal/utils.go) menghapus karakter awalan `!`, `/`, atau `#`.
4. **Routing Handler:**
   * Prefix `reminder` ➔ `reminderManager`
   * Prefix `tugas` ➔ `taskManager.HandleCommand(...)`
   * Prefix `pindah`, `kosong`, `libur`, `kuliahganti`, `jadwalganti`, `batalganti` ➔ `overrideManager.HandleCommand(...)`
   * Perintah jadwal reguler (`!hari ini`, `!besok`, `!next`, `!matkul`, dll.) ➔ `jadwalData.ProcessMessage(...)`
5. **Eksekusi Balasan Terpadu (`replyWithTyping`):**
   Fungsi terpusat `replyWithTyping` di [main.go](file:///f:/Project/bot-jadwal/main.go) secara otomatis menangani:
   * Reaksi emoji (`BuildReaction`) pada pesan pengguna (cth: `📝`, `🔄`, `📅`, `⏰`).
   * Simulasi status *"sedang mengetik..."* (*composing* ➔ *sleep* ➔ *paused*) agar interaksi manusiawi dan aman.
   * Pengiriman teks balasan via `client.SendMessage` beserta pencatatan log konsol.

---

## 5. 🛠️ Panduan Menambah Perintah Baru (*How to Add a Command*)

Jika ingin menambahkan sub-perintah baru (misal: `!link` atau `!info`):

### Langkah 1: Buat Handler di File Modul yang Sesuai
Jika berhubungan dengan jadwal/informasi, tambahkan logika di [schedule.go](file:///f:/Project/bot-jadwal/schedule.go) pada fungsi `ProcessMessage`:
```go
case "link", "drive":
    return j.GetClassLinks()
```

### Langkah 2: Daftarkan Kata Kunci
Tambahkan kata kunci pada daftar `GetKeywords()` di [schedule.go](file:///f:/Project/bot-jadwal/schedule.go) agar terdokumentasi di menu `!keyword` bot.

### Langkah 3: Tambahkan Unit Test
Buka file `_test.go` terkait (misal [schedule_test.go](file:///f:/Project/bot-jadwal/schedule_test.go)) dan tambahkan skenario pengujian untuk memastikan perintah merespons string yang diharapkan.

Jalankan pengujian:
```bash
go test -v .
```

---

## 6. 🧪 Rangkaian Pengujian Otomatis (*Testing Standards*)

Seluruh perubahan kode **wajib** lolos pengujian tanpa kegagalan sebelum di-deploy:
* `TestUtils`: Menguji parser tanggal alami, ekstraksi jam, dan lokalisasi Indonesia.
* `TestSchedule`: Menguji pemuatan JSON, pencarian dosen/ruangan/matkul, dan kalkulasi `!next`.
* `TestOverrideManager`: Menguji pemindahan jadwal, peniadaan kelas, bentrok jadwal, dan hari libur.
* `TestTaskManager`: Menguji operasi CRUD tugas, tenggat waktu, hak admin, filter matkul, dan arsip riwayat.
