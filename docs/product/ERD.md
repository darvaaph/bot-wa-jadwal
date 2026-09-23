# Entity Relationship Diagram Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 2.0.0 |
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
    classes ||--o{ roleAssignments : scopes
    classes ||--o{ semesters : contains
    semesters ||--o{ courseOfferings : offers
    courses ||--o{ courseOfferings : instantiates
    courseOfferings ||--o{ schedulePatterns : schedules
    classes ||--o{ teachingEvents : owns
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
    users ||--o{ recoveryTokens : requests
    classes o|..o{ roleAssignments : scopes
    courseOfferings o|..o{ roleAssignments : assigns
    roleInvitations ||..o| roleAssignments : activates
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
        string token_hash UK
        datetime last_seen_at
        datetime expires_at
        datetime revoked_at
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
- `KM` memakai scope `CLASS`; masa jabatan memakai `valid_from` dan `valid_until`, bukan `semester_id`.
- `PJ` memakai scope `COURSE_OFFERING` dan wajib memiliki kelas, semester, serta offering yang konsisten.
- Undangan memakai `PENDING`, `ACCEPTED`, `EXPIRED`, atau `REVOKED`; role assignment terpisah memakai `ACTIVE`, `SUSPENDED`, atau `REVOKED`.

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
    classes ||--o{ teachingEvents : owns
    schedulePatterns o|..o{ teachingEvents : originates
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
        int owner_class_id FK
        int origin_schedule_pattern_id FK
        date origin_occurrence_date
        int result_schedule_pattern_id FK
        string event_kind
        datetime starts_at
        datetime ends_at
        int room_id FK
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
        datetime confirmed_at
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

`event_kind` memakai `REPLACEMENT`, `EXTRA`, `HOLIDAY`, atau `SESSION_CANCELLED`. Lifecycle publikasi memakai `DRAFT`, `PUBLISHED`, atau `REVOKED`. Setiap event memiliki satu offering pemilik; offering peserta baru terlihat setelah KM kelasnya menerima partisipasi. Konfirmasi TU terikat pada event draf dan tetap dilakukan di luar aplikasi.

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
    users ||--o{ materials : creates

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
        string status
        string review_state
        int created_by_user_id FK
        datetime archived_at
        int version
        datetime deleted_at
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
        string visibility
        string status
        int created_by_user_id FK
        int version
        datetime deleted_at
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

Pengarsipan tugas memakai `archived_at` dan tidak mengganti status hasil. Review KM tersimpan append-only pada `task_reviews`. Materi selalu memiliki kelas; `course_offering_id` dan `task_id` bersifat opsional untuk materi umum.

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

### 2.0.0, 23 September 2026

- Menambah sesi portal dan memisahkan undangan dari role assignment.
- Mengganti jadwal lama dengan pola, teaching event, dan junction lintas kelas.
- Menambah task review, materi umum kelas, batas semester, dan konteks role pada audit.
