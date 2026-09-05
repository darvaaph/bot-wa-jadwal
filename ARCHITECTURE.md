# 🏗️ Arsitektur Sistem & Panduan Pengembang (*Architecture & Maintainer Guide*)

Dokumen ini ditujukan bagi pengembang (*maintainer / contributor*) untuk memahami arsitektur internal, struktur modularitas, skema basis data, kehandalan konkurensi, alur pemrosesan pesan, dan panduan penambahan fitur pada **Bot WhatsApp Jadwal Kuliah & Manajemen Tugas**.

---

## 1. 🌐 Arsitektur Tingkat Tinggi (*High-Level Architecture*)

Aplikasi dibangun menggunakan **Go (Golang)** dengan pola modular berbobot ringan (*monolithic lightweight service*) berbasis **Pure Go** (bebas dependensi CGO/GCC). Hal ini memungkinkan kompilasi lintas platform yang sangat cepat dan distribusi biner tunggal (*single binary*).

```mermaid
graph TD
    WA[WhatsApp Server] <-->|Websocket E2E TLS| WM[whatsmeow Client]
    
    subgraph "Aplikasi Bot (Go Runtime)"
        WM -->|Message Events| EH[Event Handler & Dispatcher<br/>main.go]
        
        EH -->|Jadwal & Pencarian| SC[Schedule Engine<br/>schedule.go]
        EH -->|Override / Pindah / Libur| OM[Override Manager<br/>override.go]
        EH -->|Deadline & Filter Tugas| TM[Task Manager<br/>task.go]
        EH -->|Pengingat Otomatis| RM[Reminder Manager<br/>reminder.go]
        
        EH -->|Sentralisasi Balasan| RWT[replyWithTyping<br/>main.go]
        RWT -->|Reaction + Presence + Reply| WM
        
        SCHED[Background Cron 06:30 WIB<br/>reminder.go] -->|Broadcast Pagi| WM
        WD[Watchdog Supervisor<br/>main.go] -.->|Auto-Reconnect Backoff| WM
        
        SC & OM & TM & RM -.-> UT[Shared Utilities<br/>utils.go]
        TM -->|Shared *sql.DB| DBP[SQLite Pool WAL Mode<br/>db.go]
        OM -->|Shared *sql.DB| DBP
    end

    subgraph "Penyimpanan Data (Storage & Persistence)"
        SC -->|Read/Reload| JSN[(jadwal.json)]
        RM -->|Read/Write JSON| RJ[(reminder_groups.json)]
        DBP <-->|Read/Write WAL| DB[(tugas.db)]
        WM -->|Device Session| SDB[(sesi_bot.db)]
    end
```

---

## 2. 📂 Peta Direktori & Tanggung Jawab Modul (*Module Directory Map*)

Seluruh logika utama berada di dalam `package main` untuk menjaga kesederhanaan eksekusi, menghindari siklus import (*cyclic dependency*), dan mempermudah pemeliharaan:

| Berkas | Tanggung Jawab Utama | Rangkaian Uji Terkait |
| :--- | :--- | :--- |
| [main.go](file:///f:/Project/bot-jadwal/main.go) | Titik masuk utama (*entry point*), inisialisasi whatsmeow, dispatcher event pesan, helper respons terpadu `replyWithTyping`, supervisor rekoneksi (*watchdog*), dan pembersihan aman (*graceful shutdown*). | Manual / Integration |
| [class_manager.go](file:///f:/Project/bot-jadwal/class_manager.go) | **Multi-Class Schedule Engine (`ClassManager`)**: Memindai direktori `data/jadwal/*.json`, mengelola in-memory cache jadwal multi-kelas, normalisasi case-insensitive, propagasi `OverrideManager`, dan hot-reload seluruh kelas (`ReloadAll`). | [class_manager_test.go](file:///f:/Project/bot-jadwal/class_manager_test.go) |
| [chat_settings.go](file:///f:/Project/bot-jadwal/chat_settings.go) | **Multi-Tenant Chat Binding (`ChatSettingsManager`)**: Mengelola preferensi grup/chat (tabel `chat_settings`), write-through cache in-memory, dan handler perintah kelas (`!kelas`, `!setkelas`, `!resetkelas`). | [chat_settings_test.go](file:///f:/Project/bot-jadwal/chat_settings_test.go) |
| [db.go](file:///f:/Project/bot-jadwal/db.go) | **Unified SQLite Connection Pool (`InitDB`)**: Sentralisasi koneksi `*sql.DB` bersama dengan mode WAL (`journal_mode=WAL`), `busy_timeout=5000`, dan `foreign_keys=1` guna menjamin nol persaingan penguncian database pada Windows. | [db_test.go](file:///f:/Project/bot-jadwal/db_test.go) |
| [utils.go](file:///f:/Project/bot-jadwal/utils.go) | **Single Source of Truth** untuk helper: lokalisasi hari/bulan Indonesia, parser tanggal alami (*relative date parser*), kalkulasi rentang jam, pembersih prefix perintah, dan pembacaan flexible time SQLite. | [utils_test.go](file:///f:/Project/bot-jadwal/utils_test.go) |
| [schedule.go](file:///f:/Project/bot-jadwal/schedule.go) | Engine jadwal kuliah: parsing kurikulum JSON, pencarian cerdas/alias (*fuzzy match*), kalkulasi kuliah aktif/berikutnya (`!next`), tampilan menu bot (`!menu`), dan kamus bantuan (`!keyword`). | [schedule_test.go](file:///f:/Project/bot-jadwal/schedule_test.go) |
| [override.go](file:///f:/Project/bot-jadwal/override.go) | Engine jadwal pengganti sementara: perubahan jam (`!pindah`), pembatalan kelas (`!kosong`), kuliah pengganti (`!kuliahganti`), pengumuman hari libur (`!libur`), deteksi bentrok jadwal, dan pembatalan (`!batalganti`). | [override_test.go](file:///f:/Project/bot-jadwal/override_test.go) |
| [task.go](file:///f:/Project/bot-jadwal/task.go) | Engine pelacak tugas SQLite: CRUD catatan tugas (`!tugas`), validasi matkul resmi, filter per mata kuliah (`!tugas sbd`), perpanjangan tenggat (`!tugas edit`), badge urgensi, dan riwayat tugas selesai (`!tugas riwayat`). | [task_test.go](file:///f:/Project/bot-jadwal/task_test.go) |
| [reminder.go](file:///f:/Project/bot-jadwal/reminder.go) | Scheduler latar belakang: pengingat otomatis pagi (06:30 WIB) setiap Senin-Jumat dengan personalisasi jadwal per kelas grup, pemformatan pesan harian terintegrasi tugas mendesak, dan daftar grup di `reminder_groups.json`. | [reminder_test.go](file:///f:/Project/bot-jadwal/reminder_test.go) |

---

## 3. 🗄️ Skema Basis Data & Pola Persistensi

### A. Tabel `tasks` (Database: `tugas.db`)
Menyimpan seluruh catatan tugas kelas (lingkup grup) maupun catatan tugas mandiri (lingkup DM pribadi):

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_jid TEXT NOT NULL,          -- JID grup (cth: 120363xxx@g.us) atau JID user (cth: 62812xxx@s.whatsapp.net)
    is_group BOOLEAN NOT NULL,         -- 1 jika di grup kelas, 0 jika di DM pribadi
    matkul TEXT NOT NULL,             -- Nama resmi matkul (cth: SISTEM BASIS DATA)
    deskripsi TEXT NOT NULL,          -- Detail instruksi / deskripsi tugas
    deadline TEXT NOT NULL,           -- Label teks tenggat seragam (cth: "Jumat, 11 Sep 23:59 WIB")
    deadline_at DATETIME,             -- Timestamp UTC/Local untuk sorting dan kalkulasi badge urgensi
    created_by TEXT NOT NULL,         -- JID pengguna yang membuat catatan tugas
    is_done BOOLEAN DEFAULT 0,        -- 0 = aktif, 1 = selesai (diarsipkan ke riwayat)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tasks_scope ON tasks(scope_jid, is_done);
```

### B. Tabel `schedule_overrides` (Database: `tugas.db`)
Menyimpan perubahan jadwal perkuliahan yang hanya berlaku mengikat pada tanggal spesifik (*Date-Specific Override*):

```sql
CREATE TABLE IF NOT EXISTS schedule_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_jid TEXT NOT NULL,          -- JID grup kelas
    override_type TEXT NOT NULL,      -- RESCHEDULE, CANCEL, EXTRA, HOLIDAY
    kode_matkul TEXT NOT NULL,        -- Kode matkul kurikulum (atau "LIBUR")
    nama_matkul TEXT NOT NULL,        -- Nama mata kuliah (atau nama libur)
    dosen TEXT NOT NULL,              -- Nama pengajar
    inisial_dosen TEXT NOT NULL,      -- Inisial pengajar
    orig_date TEXT NOT NULL,          -- YYYY-MM-DD tanggal asal jadwal normal
    orig_jam TEXT NOT NULL,           -- Jam asal (cth: "07:00 - 08:40")
    target_date TEXT NOT NULL,        -- YYYY-MM-DD tanggal berlakunya perubahan
    new_jam TEXT NOT NULL,            -- Jam baru (cth: "13:00 - 14:40")
    ruang TEXT NOT NULL,              -- Ruangan baru
    alasan TEXT NOT NULL,             -- Keterangan / alasan pergeseran
    created_by TEXT NOT NULL,         -- JID Komti / Admin yang memodifikasi
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_overrides_scope ON schedule_overrides(scope_jid, target_date);
```

### C. Tabel `chat_settings` (Database: `tugas.db`)
Menyimpan binding relasi antara grup WhatsApp / chat pribadi terhadap jadwal kelas tertentu (*Multi-Tenant Support*):

```sql
CREATE TABLE IF NOT EXISTS chat_settings (
    scope_jid TEXT PRIMARY KEY,       -- JID grup (120363xxx@g.us) atau JID user
    class_id TEXT NOT NULL,           -- ID kelas huruf kapital (contoh: "3A", "3B")
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### D. File Konfigurasi & Master Data

#### 1. Direktori Master Jadwal `data/jadwal/*.json`
Menyimpan master data kurikulum modular per kelas (misal: `data/jadwal/3a.json`, `data/jadwal/3b.json`). Format berkas mempertahankan schema `JadwalConfig`: kampus, daftar dosen, kode matkul, dan jadwal perkuliahan. Bot juga mendukung *fallback* membaca `jadwal.json` di root jika direktori belum dibuat.

#### 2. `reminder_groups.json`
Menyimpan konfigurasi pengingat otomatis dan daftar grup penerima:
```json
{
  "hour": 6,
  "minute": 30,
  "groups": [
    {
      "jid": "120363001234567890@g.us",
      "name": "D4 Teknik Informatika 3A",
      "added_at": "2026-09-04T08:00:00Z"
    }
  ]
}
```

---

## 4. ⚡ Manajemen Konkurensi & Kehandalan (*Reliability & Concurrency*)

### A. Shared SQLite Connection Pool ([db.go](file:///f:/Project/bot-jadwal/db.go))
* **Masalah Lama:** SQLite di Windows rentan memunculkan error `database is locked` jika terdapat lebih dari satu connection pool yang mengakses file fisik yang sama.
* **Solusi Terpadu:** Modul [db.go](file:///f:/Project/bot-jadwal/db.go) menginisialisasi satu instance `*sql.DB` bersama (`InitDB`) dan menginjeksinya ke `TaskManager` dan `OverrideManager`.
* **Parameter PRAGMA Optimal:**
  * `journal_mode=WAL`: Mengizinkan pembaca (*readers*) dan satu penulis (*writer*) bekerja bersamaan tanpa saling memblokir.
  * `busy_timeout=5000`: Menunggu hingga 5.000 ms jika ada proses tulis sebelum mengembalikan error.
  * `synchronous=NORMAL`: Kecepatan operasi disk optimal tanpa mengorbankan durabilitas data WAL.

### B. Auto-Reconnect Watchdog Supervisor
Koneksi websocket kampus/kosan rentan mengalami *EOF/RST*. Bot dilengkapi supervisor latar belakang di [main.go](file:///f:/Project/bot-jadwal/main.go):
* Memantau status `client.IsConnected()` secara berkala.
* Menerapkan **Exponential Backoff** (3s ➔ 6s ➔ 12s ➔ maks 30s) untuk menghindari *spamming reconnection*.

### C. Graceful Shutdown Terstruktur
Saat menerima sinyal terminasi (`Ctrl + C` / `SIGTERM`), bot mengeksekusi urutan pembersihan berjenjang:
1. Membatalkan konteks goroutine watchdog supervisor.
2. Memutuskan koneksi WhatsApp klien secara teratur (`client.Disconnect()`).
3. Menutup koneksi database aplikasi (`appDB.Close()`) untuk melakukan sinkronisasi (*checkpoint*) file WAL SQLite.
4. Menutup koneksi database sesi bot (`container.Close()`).

---

## 5. 🔐 Otorisasi & Hak Akses Berbasis Lingkup (*Role & Scope-Based Authorization*)

Bot menerapkan pemisahan hak akses yang ketat untuk menjaga integritas data kelas:

| Lingkup Obrolan | Tipe Perintah | Hak Akses | Logika Verifikasi |
| :--- | :--- | :--- | :--- |
| **Grup Kelas (`@g.us`)** | Modifikasi Jadwal (`!pindah`, `!kosong`, `!kuliahganti`, `!libur`, `!batalganti`) | **Khusus Admin Grup** | Diperiksa via fungsi `isSenderGroupAdmin` yang mencocokkan JID pengirim dengan daftar admin grup WhatsApp. |
| **Grup Kelas (`@g.us`)** | Modifikasi Tugas (`!tugas tambah`, `!tugas hapus`, `!tugas edit`) | **Khusus Admin Grup** | Melindungi catatan tugas kelas dari penghapusan/perubahan oleh anggota biasa. |
| **Grup Kelas (`@g.us`)** | Pembacaan (`!menu`, `!hari ini`, `!tugas`, `!next`, `!jadwalganti`, dll.) | **Semua Anggota** | Terbuka bebas untuk seluruh mahasiswa dalam grup. |
| **Chat Pribadi (`@s.whatsapp.net`)** | Semua Perintah | **Bebas (Pribadi)** | Setiap nomor WhatsApp otomatis menjadi admin untuk database catatan tugas pribadinya sendiri (*Isolated User Scope*). |

---

## 6. 🔄 Siklus Pemrosesan Pesan (*Message Processing Lifecycle*)

```
[Pesan Masuk WhatsApp]
       │
       ▼
1. Filter Awal (Abaikan jika v.Info.IsFromMe == true)
       │
       ▼
2. Ekstraksi Teks (Conversation / ExtendedTextMessage)
       │
       ▼
3. Pembersihan Awalan (cleanCommandPrefix: hapus '!', '/', atau '#')
       │
       ▼
4. Dispatching Handler:
   ├── Prefix 'reminder'       ──► ReminderManager
   ├── Prefix 'tugas'          ──► TaskManager.HandleCommand(...)
   ├── Prefix 'pindah/kosong/
   │          libur/override'  ──► OverrideManager.HandleCommand(...)
   └── Perintah Jadwal Reguler ──► JadwalConfig.ProcessMessage(...)
       │
       ▼
5. Eksekusi Balasan Terpadu via replyWithTyping(...)
   ├── Berikan Emoji Reaction pada pesan pengguna (cth: ⏰, 📝, 🔄, 📅)
   ├── Kirim Chat Presence "Sedang Mengetik..." (Composing -> Sleep -> Paused)
   └── Kirim Pesan Balasan via client.SendMessage & Catat Log Konsol
```

---

## 7. 🛠️ Panduan Menambahkan Perintah Baru (*How to Add a Command*)

Untuk menambahkan perintah baru (misal: `!link` untuk direktori link Google Meet / Drive materi kuliah):

### Langkah 1: Implementasikan Handler di Modul Terkait
Tambahkan fungsi penanganan pada modul yang sesuai, misalnya di [schedule.go](file:///f:/Project/bot-jadwal/schedule.go):
```go
// GetClassLinks mengembalikan direktori tautan perkuliahan
func (j *JadwalConfig) GetClassLinks() string {
    return "🔗 *DIREKTORI TAUTAN KELAS*\n• Google Drive: https://bit.ly/drive-kelas\n• Grup Pengumuman: ..."
}
```

### Langkah 2: Daftarkan Perintah pada Parser Pesan
Pada method `ProcessMessage` di [schedule.go](file:///f:/Project/bot-jadwal/schedule.go), tambahkan percabangan:
```go
case "link", "drive", "tautan":
    return j.GetClassLinks()
```

### Langkah 3: Daftarkan di Kamus Bantuan & Menu
Tambahkan kata kunci pada method `GetKeywords()` dan format template di `GetMenu()` agar mahasiswa dapat menemukan perintah tersebut.

### Langkah 4: Buat Unit Test Otomatis
Buka berkas uji terkait (misal [schedule_test.go](file:///f:/Project/bot-jadwal/schedule_test.go)) dan tambahkan skenario validasi:
```go
reply := cfg.ProcessMessage("!link", false, "")
if !strings.Contains(reply, "DIREKTORI TAUTAN KELAS") {
    t.Errorf("Format balasan !link tidak sesuai: %s", reply)
}
```

### Langkah 5: Jalankan Pengujian
```bash
go test -count=1 -v .
```

---

## 8. 🧪 Standar Rangkaian Pengujian (*Testing Standards*)

Rangkaian unit test wajib mencakup skenario sukses (*positive test*), skenario kegagalan/penolakan (*negative test*), pengujian batas waktu (*edge cases*), dan konkurensi:

| Test Suite | Berkas Uji | Cakupan Pengujian |
| :--- | :--- | :--- |
| `TestSharedSQLiteConnection` | [db_test.go](file:///f:/Project/bot-jadwal/db_test.go) | Inisialisasi pool SQLite, verifikasi WAL mode, dan konkurensi penulisan paralel simultan 20 goroutine antara `TaskManager` & `OverrideManager`. |
| `TestUtils` | [utils_test.go](file:///f:/Project/bot-jadwal/utils_test.go) | Validasi nama hari/bulan Indonesia, ekstraksi kata tanggal, kalkulasi rentang jam, dan pembersihan awalan prefix. |
| `TestSchedule` | [schedule_test.go](file:///f:/Project/bot-jadwal/schedule_test.go) | Parsing kurikulum `jadwal.json`, pencarian cerdas mata kuliah/dosen/ruangan, kalkulasi `!next`, dan perpaduan jadwal reguler dengan override. |
| `TestOverrideManager` | [override_test.go](file:///f:/Project/bot-jadwal/override_test.go) | Penjadwalan ulang (`!pindah`), pembatalan kelas (`!kosong`), deteksi bentrok jadwal, pengumuman hari libur (`!libur`), dan pembatalan (`!batalganti`). |
| `TestTaskManager` | [task_test.go](file:///f:/Project/bot-jadwal/task_test.go) | CRUD tugas SQLite, otorisasi admin grup vs anggota biasa, filter per mata kuliah (`!tugas sbd`), perpanjangan tenggat (`!tugas edit`), badge urgensi, dan riwayat tugas selesai (`!tugas riwayat`). |
