# Entity Relationship Diagram Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 3.0.2 |
| Status | Approved |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |
| Model sumber | [Data Model](DATA_MODEL.md) |

Dokumen ini memvisualisasikan target logical data model Bot Jadwal. Diagram dapat dirender pada Markdown yang mendukung Mermaid atau disalin ke [Mermaid Live Editor](https://mermaid.live). Detail constraint, status, indeks, dan strategi migrasi tetap mengikuti `DATA_MODEL.md`.

## 1. Legenda

| Notasi | Arti |
|---|---|
| `PK` | Primary key |
| `FK` | Foreign key |
| `UK` | Unique key |
| Garis penuh | Relasi utama atau identifying |
| Garis putus-putus | Relasi opsional, lemah, atau referensi logis |
| `||` | Tepat satu |
| `o|` | Nol atau satu |
| `|{` | Satu atau lebih |
| `o{` | Nol atau lebih |

Kolom audit umum seperti `created_at`, `updated_at`, `deleted_at`, dan `version` tidak selalu ditampilkan agar diagram tetap terbaca.

## 2. Overview Domain

Diagram ini menunjukkan tulang punggung model tanpa seluruh atribut.

```mermaid
erDiagram
    direction LR

    users ||--o{ roleAssignments : receives
    users o|..o{ loginAttempts : may_match
    users ||--o{ userSessions : owns
    classes o|..o{ roleAssignments : scopes
    classes ||--o{ semesters : contains
    semesters ||--o{ courseOfferings : offers
    courses ||--o{ courseOfferings : instantiates
    courseOfferings ||--o{ schedulePatterns : schedules
    classes o|..o{ teachingEvents : derived_owner
    teachingEvents ||--|{ teachingEventOfferings : includes
    courseOfferings ||--o{ teachingEventOfferings : participates
    courseOfferings ||--o{ tasks : owns
    tasks ||--o{ taskReviews : receives
    classes ||--o{ materials : owns
    classes ||--o{ whatsappChannels : uses
    whatsappChannels ||--o{ notificationMessages : receives
    classes o|..o{ auditLogs : scopes
    users o|..o{ auditLogs : acts_in

    users["users"]
    roleAssignments["role_assignments"]
    loginAttempts["login_attempts"]
    userSessions["user_sessions"]
    classes["classes"]
    semesters["semesters"]
    courses["courses"]
    courseOfferings["course_offerings"]
    schedulePatterns["schedule_patterns"]
    teachingEvents["teaching_events"]
    teachingEventOfferings["teaching_event_offerings"]
    tasks["tasks"]
    taskReviews["task_reviews"]
    materials["materials"]
    whatsappChannels["whatsapp_channels"]
    notificationMessages["notification_messages"]
    auditLogs["audit_logs"]
```

## 3. Identitas dan Akses

```mermaid
erDiagram
    direction LR

    users ||--o{ roleAssignments : receives
    users ||--o{ roleInvitations : creates
    users ||--o{ userSessions : owns
    users o|..o{ loginAttempts : may_match
    roleAssignments o|..o{ userSessions : active_context
    users ||--o{ recoveryTokens : requests
    classes o|..o{ roleAssignments : scopes
    semesters o|..o{ roleAssignments : limits
    courseOfferings o|..o{ roleAssignments : assigns
    roleInvitations o|..o| roleAssignments : activates
    classes o|..o{ roleInvitations : scopes
    semesters o|..o{ roleInvitations : limits
    courseOfferings o|..o{ roleInvitations : assigns
    classes ||--o{ portalSessions : grants

    users["users"] {
        int id PK
        string identity_key UK
        string display_name
        string password_hash
        string status
        int session_version
        datetime last_login_at
    }
    roleAssignments["role_assignments"] {
        int id PK
        int user_id FK
        string role
        string scope_type
        int class_id FK
        int semester_id FK
        int course_offering_id FK
        int accepted_invitation_id FK
        string status
        datetime valid_from
        datetime valid_until
        int version
    }
    roleInvitations["role_invitations"] {
        int id PK
        string token_hash UK
        string invited_identity_key
        string role
        string scope_type
        int class_id FK
        int semester_id FK
        int course_offering_id FK
        int invited_by_user_id FK
        string status
        datetime expires_at
    }
    userSessions["user_sessions"] {
        int id PK
        int user_id FK
        int active_role_assignment_id FK
        string token_hash UK
        int session_version
        datetime created_at
        datetime last_seen_at
        datetime absolute_expires_at
        datetime revoked_at
        string revocation_reason
    }
    loginAttempts["login_attempts"] {
        int id PK
        int user_id FK
        string identity_hash
        string source_hash
        string outcome
        datetime attempted_at
    }
    portalSessions["portal_sessions"] {
        int id PK
        int class_id FK
        string token_hash UK
        int access_code_version
        datetime expires_at
        datetime revoked_at
    }
    recoveryTokens["recovery_tokens"] {
        int id PK
        int user_id FK
        string token_hash UK
        string method
        datetime expires_at
        datetime used_at
    }
    classes["classes"] {
        int id PK
        string code UK
        string slug UK
    }
    semesters["semesters"] {
        int id PK
        int class_id FK
        string status
    }
    courseOfferings["course_offerings"] {
        int id PK
        int semester_id FK
        int course_id FK
    }
```

Aturan scope:

- `SYSTEM_ADMIN` memakai scope `GLOBAL` tanpa foreign key kelas.
- Assignment hasil provisioning, termasuk System Admin pertama, boleh tidak memiliki `accepted_invitation_id`; assignment dari undangan wajib menunjuk undangan yang diterima.
- `KM` memakai scope `CLASS`; masa jabatan memakai `valid_from` dan `valid_until`, bukan `semester_id`.
- `PJ` memakai scope `COURSE_OFFERING` dan wajib memiliki kelas, semester, serta offering yang konsisten.
- Undangan memakai `PENDING`, `ACCEPTED`, `EXPIRED`, atau `REVOKED`; role assignment terpisah memakai `ACTIVE`, `SUSPENDED`, atau `REVOKED`.
- Sesi menunjuk satu role assignment aktif. Pergantian konteks merotasi token; `session_version` mencabut seluruh sesi pengguna saat diperlukan.
- `login_attempts` menyimpan hash identitas dan sumber, bukan kredensial atau alamat sumber mentah.

## 4. Struktur Akademik

```mermaid
erDiagram
    direction LR

    classes ||--|| classSettings : configures
    classes ||--o{ semesters : contains
    semesters ||--o{ courseOfferings : offers
    courses ||--o{ courseOfferings : instantiates
    courseOfferings ||--o{ offeringLecturers : has
    lecturers ||--o{ offeringLecturers : teaches

    classes["classes"] {
        int id PK
        string code UK
        string slug UK
        string study_program
        int cohort_year
        string group_label
        string status
    }
    classSettings["class_settings"] {
        int class_id PK, FK
        string timezone
        string portal_access_mode
        string portal_code_hash
        int portal_code_version
        string meeting_link_visibility
        string morning_reminder_time
        string afternoon_reminder_time
        int replacement_reminder_minutes
        int version
    }
    semesters["semesters"] {
        int id PK
        int class_id FK
        string academic_year
        string term
        date starts_on
        date ends_on
        string status
        datetime published_at
        datetime activated_at
        datetime archived_at
        int version
    }
    courses["courses"] {
        int id PK
        string code UK
        string name
        string status
    }
    courseOfferings["course_offerings"] {
        int id PK
        int semester_id FK
        int course_id FK
        string activity_type
        string display_name
        string status
        int version
    }
    lecturers["lecturers"] {
        int id PK
        string code UK
        string full_name
        string status
    }
    offeringLecturers["offering_lecturers"] {
        int course_offering_id PK, FK
        int lecturer_id PK, FK
        string responsibility
    }
```

Satu kelas hanya memiliki satu semester `ACTIVE`. Teori dan praktikum dapat menjadi course offering berbeda agar jadwal, dosen, serta PJ dapat dikelola secara terpisah.

## 5. Jadwal dan Ruangan

```mermaid
erDiagram
    direction LR

    courseOfferings ||--o{ schedulePatterns : schedules
    rooms o|..o{ schedulePatterns : locates
    schedulePatterns o|..o{ teachingEvents : originates
    teachingEvents o|..o| schedulePatterns : results_in
    teachingEvents ||--|{ teachingEventOfferings : includes
    courseOfferings ||--o{ teachingEventOfferings : participates
    rooms o|..o{ teachingEvents : locates
    teachingEvents ||--o{ roomConfirmations : requests
    rooms ||--o{ roomConfirmations : confirms
    users ||--o{ roomConfirmations : records

    courseOfferings["course_offerings"] {
        int id PK
        int semester_id FK
        int course_id FK
    }
    rooms["rooms"] {
        int id PK
        string code UK
        string name
        string building
        string room_type
        int capacity
        string status
        datetime source_updated_at
    }
    schedulePatterns["schedule_patterns"] {
        int id PK
        int course_offering_id FK
        int room_id FK
        int day_of_week
        string start_time
        string end_time
        date effective_from
        date effective_until
        string status
        int version
    }
    teachingEvents["teaching_events"] {
        int id PK
        int origin_schedule_pattern_id FK
        date origin_occurrence_date
        int result_schedule_pattern_id FK
        string event_kind
        datetime starts_at
        datetime ends_at
        int room_id FK
        string reason
        string conflict_override_reason
        string lifecycle_status
        int published_by_user_id FK
        datetime published_at
        int revoked_by_user_id FK
        datetime revoked_at
        string revocation_reason
        int version
    }
    teachingEventOfferings["teaching_event_offerings"] {
        int teaching_event_id PK, FK
        int course_offering_id PK, FK
        string participation_role
        string participation_status
        int responded_by_user_id FK
        datetime responded_at
    }
    roomConfirmations["room_confirmations"] {
        int id PK
        int teaching_event_id FK
        int room_id FK
        string confirmation_status
        string external_contact
        string note
        int recorded_by_user_id FK
        datetime recorded_at
        datetime confirmed_at
    }
    users["users"] {
        int id PK
        string display_name
    }
```

`event_kind` memakai `REPLACEMENT`, `EXTRA`, `HOLIDAY`, atau `SESSION_CANCELLED`. Lifecycle publikasi memakai `DRAFT`, `PUBLISHED`, atau `REVOKED`. Kelas pemilik diturunkan dari tepat satu offering `OWNER`. Offering peserta harus berasal dari kelas lain dan baru terlihat setelah KM kelasnya menerima partisipasi. Konfirmasi TU terikat pada event draf; `recorded_at` selalu terisi, sedangkan `confirmed_at` hanya terisi untuk hasil `CONFIRMED`.

## 6. Tugas dan Materi

```mermaid
erDiagram
    direction LR

    courseOfferings ||--o{ tasks : owns
    tasks ||--o{ taskReviews : receives
    users ||--o{ taskReviews : writes
    classes ||--o{ materials : owns
    courseOfferings o|..o{ materials : scopes
    tasks o|..o{ materials : references
    users ||--o{ tasks : creates
    users o|..o{ tasks : deletes
    users ||--o{ materials : creates
    users o|..o{ materials : deletes

    courseOfferings["course_offerings"] {
        int id PK
        int semester_id FK
        int course_id FK
    }
    tasks["tasks"] {
        int id PK
        int course_offering_id FK
        string title
        text instructions
        datetime deadline_at
        string task_type
        string submission_text
        string submission_url
        string publication_status
        string review_state
        int reviewed_version
        int created_by_user_id FK
        datetime published_at
        datetime completed_at
        datetime archived_at
        int version
        datetime deleted_at
        int deleted_by_user_id FK
    }
    taskReviews["task_reviews"] {
        int id PK
        int task_id FK
        int reviewer_user_id FK
        int reviewer_role_assignment_id FK
        int task_version
        string decision
        string note
        datetime created_at
    }
    materials["materials"] {
        int id PK
        int class_id FK
        int course_offering_id FK
        int task_id FK
        string title
        string material_type
        string url
        text description
        string visibility
        string status
        int created_by_user_id FK
        int version
        datetime deleted_at
        int deleted_by_user_id FK
    }
    users["users"] {
        int id PK
        string display_name
    }
    classes["classes"] {
        int id PK
        string code UK
    }
```

Publikasi tugas memakai `DRAFT`, `PUBLISHED`, atau `REVOKED`. Penyelesaian memakai `completed_at`; keterlambatan dihitung dari deadline. Pengarsipan memakai `archived_at`. Soft delete mengisi `deleted_at` dan `deleted_by_user_id`. Review KM tersimpan append-only dan hanya berlaku untuk `task_version` yang sama dengan versi tugas. Materi selalu memiliki kelas; `course_offering_id` dan `task_id` bersifat opsional untuk materi umum.

## 7. WhatsApp dan Notifikasi

```mermaid
erDiagram
    direction LR

    classes ||--o{ whatsappChannels : uses
    whatsappChannels ||--o{ notificationMessages : receives
    notificationMessages ||--o{ notificationAttempts : retries
    notificationMessages o|..o{ notificationMessages : supersedes
    users o|..o{ notificationMessages : triggers

    classes["classes"] {
        int id PK
        string code UK
        string slug UK
    }
    whatsappChannels["whatsapp_channels"] {
        int id PK
        int class_id FK
        string jid UK
        string channel_type
        string display_name
        string status
        datetime verified_at
    }
    notificationMessages["notification_messages"] {
        int id PK
        int class_id FK
        int whatsapp_channel_id FK
        string event_type
        string entity_type
        int entity_id
        string idempotency_key UK
        json payload_json
        string status
        datetime scheduled_at
        datetime sent_at
        int supersedes_message_id FK
        int triggered_by_user_id FK
    }
    notificationAttempts["notification_attempts"] {
        int id PK
        int notification_message_id FK
        int attempt_number
        datetime started_at
        datetime finished_at
        string result
        string error_message
        string provider_message_id
    }
    users["users"] {
        int id PK
        string display_name
    }
```

Data akademik dimiliki kelas, bukan JID. Koneksi WhatsApp yang putus hanya mengubah kanal dan status pesan. `idempotency_key` mencegah satu kejadian bisnis menghasilkan pesan ganda.

## 8. Audit, Impor, dan Backup

```mermaid
erDiagram
    direction LR

    classes o|..o{ auditLogs : scopes
    semesters o|..o{ auditLogs : scopes
    users o|..o{ auditLogs : acts_in
    roleAssignments o|..o{ auditLogs : provides_context
    classes ||--o{ importBatches : imports
    semesters ||--o{ importBatches : targets
    importBatches ||--o{ importErrors : reports
    classes ||--o{ backupRecords : backs_up
    semesters o|..o{ backupRecords : limits
    users ||--o{ importBatches : creates
    users ||--o{ backupRecords : creates

    auditLogs["audit_logs"] {
        int id PK
        int class_id FK
        int semester_id FK
        int actor_user_id FK
        int actor_role_assignment_id FK
        json actor_context_json
        string actor_type
        string action
        string entity_type
        int entity_id
        json before_json
        json after_json
        string reason
        string correlation_id
        datetime created_at
    }
    importBatches["import_batches"] {
        int id PK
        int class_id FK
        int semester_id FK
        string source_type
        string source_checksum
        string status
        int created_by_user_id FK
        datetime created_at
        json summary_json
    }
    importErrors["import_errors"] {
        int id PK
        int batch_id FK
        string source_location
        string field_name
        string error_code
        string message
        string severity
    }
    backupRecords["backup_records"] {
        int id PK
        int class_id FK
        int semester_id FK
        string artifact_ref
        string checksum
        string status
        string reason
        int created_by_user_id FK
        datetime verified_at
    }
    classes["classes"] {
        int id PK
        string code UK
    }
    semesters["semesters"] {
        int id PK
        int class_id FK
        string status
    }
    users["users"] {
        int id PK
        string display_name
    }
    roleAssignments["role_assignments"] {
        int id PK
        int user_id FK
        string role
        string scope_type
    }
```

Audit log bersifat append-only. Relasi `entity_type` dan `entity_id` adalah referensi logis agar satu audit log dapat mencatat banyak jenis objek. Snapshot tidak boleh menyimpan password, token, kode portal, atau rahasia sesi.

## 9. Batas Diagram

ERD tidak menggambarkan seluruh aturan berikut karena aturan tersebut ditegakkan melalui constraint dan service:

- Hanya satu semester aktif per kelas.
- Kesesuaian kelas dan semester pada role assignment PJ.
- Tepat satu offering pemilik pada setiap teaching event dan penerimaan KM untuk kelas peserta.
- Kesesuaian tanggal event dengan semester offering pemilik dan peserta.
- Pemisahan `SESSION_CANCELLED` dari lifecycle publikasi `REVOKED`.
- Validasi transisi status.
- Optimistic locking melalui `version`.
- Soft delete dan kebijakan retensi.
- Foreign key polimorfik pada notifikasi dan audit.
- Transaksi aktivasi semester, publikasi, pembatalan, impor, dan restore.

Rincian lengkapnya tersedia pada [Data Model](DATA_MODEL.md), [Business Rules](BUSINESS_RULES.md), dan [Functional Requirements](FUNCTIONAL_REQUIREMENTS.md).

## 10. Cara Menggunakan

1. Buka bagian diagram yang ingin ditinjau.
2. Salin isi di antara `erDiagram` dan penutup code block.
3. Tempelkan ke Mermaid Live Editor jika preview Markdown tidak merender Mermaid.
4. Ekspor sebagai SVG untuk dokumentasi atau Figma.
5. Ubah sumber Mermaid dalam repository lebih dahulu agar diagram hasil ekspor tidak menjadi sumber kebenaran terpisah.

## 11. Changelog

### 3.0.2, 23 September 2026

- Menambahkan `import_batches.summary_json` agar ringkasan jumlah baris impor pada Data Model terwakili di ERD.

### 3.0.1, 23 September 2026

- Memperbaiki optionality undangan pada role assignment hasil provisioning.
- Menambahkan pelaku soft delete pada tugas dan materi serta relasinya ke pengguna.

### 3.0.0, 23 September 2026

- Menambah percobaan login dan konteks role assignment pada sesi.
- Menurunkan kelas pemilik event dari offering `OWNER` dan melengkapi relasi pola hasil.
- Memisahkan lifecycle publikasi tugas dari selesai, terlambat, arsip, dan review per versi.
- Menyelaraskan kolom `published_at` pada tasks, `reason` pada teaching_events, `description` pada materials, kardinalitas role assignment global, serta relasi semester pada role assignment.

### 2.0.0, 23 September 2026

- Menambah sesi portal dan memisahkan undangan dari role assignment.
- Mengganti jadwal lama dengan pola, teaching event, dan junction lintas kelas.
- Menambah task review, materi umum kelas, batas semester, dan konteks role pada audit.
