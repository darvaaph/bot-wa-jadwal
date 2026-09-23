# Traceability Matrix Bot Jadwal

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.0.0 |
| Status | Approved |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 23 September 2026 |

Dokumen ini menghubungkan kebutuhan pengguna dengan requirement kanonis, aturan bisnis, alur, dan entitas target. Hierarki sumber kebenaran adalah Product Definition, User Requirements, Functional Requirements, Business Rules dan User Flows, lalu Data Model dan ERD. ID `FR-*` hanya didefinisikan dalam `FUNCTIONAL_REQUIREMENTS.md`.

## Matriks

| User Requirement | Functional Requirement | Business Rule | User Flow | Entitas Utama |
|---|---|---|---|---|
| UR-ACCESS-001 | FR-ACCESS-001 | BR-CLASS-002 | UF-PORTAL-001 | `classes`, `class_settings`, `portal_sessions` |
| UR-ACCESS-002 | FR-ACCESS-002, FR-ACCESS-003 | BR-ACCESS-001, BR-ACCESS-005 | UF-ACCESS-001, UF-ACCESS-002 | `users`, `role_invitations`, `user_sessions` |
| UR-ACCESS-003 | FR-ACCESS-003, FR-ACCESS-004, FR-ACCESS-007 | BR-ACCESS-002, BR-ACCESS-003 | UF-ACCESS-002, UF-ACCESS-006 | `role_assignments`, `course_offerings` |
| UR-ACCESS-004 | FR-ACCESS-003, FR-ACCESS-004, FR-ACCESS-007 | BR-ACCESS-003, BR-CLASS-001 | UF-ACCESS-001, UF-ACCESS-006 | `role_assignments`, `classes` |
| UR-ACCESS-005 | FR-ACCESS-005 | BR-ACCESS-004 | UF-ACCESS-003, UF-ACCESS-004 | `users`, `role_assignments` |
| UR-ACCESS-006 | FR-ACCESS-006 | BR-ACCESS-003, BR-ACCESS-006 | UF-ACCESS-005 | `recovery_tokens`, `user_sessions`, `audit_logs` |
| UR-CLASS-001 | FR-CLASS-002 | BR-CLASS-001 | UF-PORTAL-001, UF-ACCESS-004 | `classes` dan seluruh FK kelas |
| UR-CLASS-002 | FR-CLASS-001 | BR-CLASS-001 | UF-ACCESS-001, UF-SEM-002 | `classes`, `semesters` |
| UR-SEM-001 | FR-SEM-001, FR-SEM-003, FR-SEM-004 | BR-SEM-001, BR-SEM-002, BR-SEM-003, BR-SEM-004 | UF-SEM-001, UF-SEM-002 | `semesters` |
| UR-SEM-002 | FR-SEM-001, FR-SEM-002 | BR-SEM-005 | UF-SEM-001, UF-SEM-002 | `semesters`, `import_batches` |
| UR-SEM-003 | FR-SEM-002, FR-SEM-003 | BR-SEM-005 | UF-SEM-001 | `import_batches`, `import_errors` |
| UR-SCH-001 | FR-SCH-001 | BR-SCH-001 | UF-PORTAL-002 | `schedule_patterns`, `teaching_events` |
| UR-SCH-002 | FR-SCH-002 | BR-SCH-001, BR-SCH-007 | UF-SCH-001 | `schedule_patterns` |
| UR-SCH-003 | FR-SCH-003, FR-SCH-006 | BR-SCH-001 | UF-SCH-002, UF-SCH-003 | `schedule_patterns`, `teaching_events` |
| UR-SCH-004 | FR-SCH-003 | BR-SCH-002 | UF-SCH-002 | `teaching_events` |
| UR-SCH-005 | FR-SCH-004 | BR-SCH-002, BR-SCH-007 | UF-SCH-002, UF-SCH-003 | `teaching_events`, `rooms` |
| UR-SCH-006 | FR-SCH-005 | BR-SCH-003, BR-SCH-004 | UF-SCH-002 | `teaching_events`, `audit_logs` |
| UR-SCH-007 | FR-SCH-007 | BR-SCH-005, BR-SCH-006 | UF-SCH-004 | `teaching_events`, `notification_messages` |
| UR-SCH-008 | FR-SCH-001, FR-SCH-003 | BR-SCH-001 | UF-PORTAL-002 | `schedule_patterns`, `teaching_events` |
| UR-SCH-009 | FR-SCH-008 | BR-SCH-008 | UF-SCH-005 | `teaching_events`, `teaching_event_offerings` |
| UR-TASK-001 | FR-TASK-001, FR-TASK-002 | BR-TASK-001 | UF-TASK-001 | `tasks` |
| UR-TASK-002 | FR-TASK-002, FR-TASK-005 | BR-TASK-001, BR-TASK-004 | UF-PORTAL-002, UF-TASK-001 | `tasks` |
| UR-TASK-003 | FR-TASK-004 | BR-TASK-002 | UF-PORTAL-002 | `tasks` |
| UR-TASK-004 | FR-TASK-004 | BR-TASK-002 | UF-PORTAL-002 | `tasks` |
| UR-TASK-005 | FR-TASK-002, FR-TASK-003 | BR-TASK-003 | UF-TASK-001, UF-TASK-002 | `tasks`, `task_reviews` |
| UR-TASK-006 | FR-TASK-006 | BR-TASK-002, BR-OPS-002 | UF-TASK-003 | `tasks`, `audit_logs` |
| UR-TASK-007 | FR-TASK-007 | BR-TASK-005 | UF-PORTAL-002, UF-TASK-003 | `materials` |
| UR-NOTIF-001 | FR-NOTIF-001, FR-NOTIF-005 | BR-NOTIF-001 sampai BR-NOTIF-003 | UF-PORTAL-002, UF-OPS-001 | `notification_messages`, `class_settings` |
| UR-NOTIF-002 | FR-NOTIF-002, FR-NOTIF-005 | BR-NOTIF-002, BR-NOTIF-003 | UF-PORTAL-002, UF-OPS-001 | `tasks`, `notification_messages` |
| UR-NOTIF-003 | FR-NOTIF-003 | BR-SCH-006, BR-NOTIF-003 | UF-SCH-002, UF-SCH-004 | `notification_messages` |
| UR-NOTIF-004 | FR-NOTIF-003 | BR-NOTIF-003 | UF-PORTAL-002, UF-OPS-001 | `notification_messages` |
| UR-NOTIF-005 | FR-NOTIF-004 | BR-NOTIF-001, BR-NOTIF-002, BR-NOTIF-004 | UF-OPS-001 | `notification_messages`, `notification_attempts` |
| UR-NOTIF-006 | FR-NOTIF-006 | BR-NOTIF-005 | UF-SCH-002, UF-OPS-001 | `teaching_events`, `notification_messages` |
| UR-ROOM-001 | FR-ROOM-001 | BR-ROOM-001 | UF-ROOM-001 | `rooms`, `schedule_patterns`, `teaching_events` |
| UR-ROOM-002 | FR-ROOM-001, FR-ROOM-002 | BR-ROOM-001, BR-ROOM-003 | UF-ROOM-001 | `teaching_events`, `room_confirmations` |
| UR-ROOM-003 | FR-ROOM-003 | BR-ROOM-002 | UF-ROOM-001 | `rooms` |
| UR-AUDIT-001 | FR-AUDIT-001, FR-AUDIT-002 | BR-AUDIT-001 | UF-TASK-003, UF-SCH-004 | `audit_logs`, `role_assignments` |
| UR-AUDIT-002 | FR-AUDIT-001, FR-AUDIT-002 | BR-SCH-005, BR-AUDIT-001 | UF-SCH-004 | `audit_logs`, `teaching_events` |
| UR-AUDIT-003 | FR-AUDIT-003 | BR-AUDIT-002 | UF-OPS-003 | `audit_logs`, `backup_records` |
| UR-UX-001 | FR-UX-001 | Aturan lintas fitur | UF-TASK-001, UF-SCH-002 | Semua form operasional |
| UR-UX-002 | FR-UX-002 | Aturan lintas fitur | Seluruh flow | Tidak spesifik |
| UR-UX-003 | FR-UX-003 | BR-ACCESS-004 | UF-ACCESS-004 | `role_assignments` |
| UR-UX-004 | FR-UX-004 | Aturan lintas fitur | Seluruh flow | Tidak spesifik |
| UR-OPS-001 | FR-OPS-001 | BR-OPS-004 | UF-OPS-001 | `whatsapp_channels`, `notification_messages` |
| UR-OPS-002 | FR-OPS-002 | BR-OPS-003 | UF-OPS-003 | `backup_records` |
| UR-OPS-003 | FR-OPS-003 | BR-OPS-002 | UF-TASK-003, UF-OPS-003 | Entitas dengan `deleted_at`, `audit_logs` |
| UR-OPS-004 | FR-OPS-004 | BR-OPS-001 | UF-OPS-002 | Entitas dengan `version` |

## Aturan Pemeliharaan

- Tambahkan baris sebelum requirement baru berstatus `Approved`.
- Jangan mendefinisikan arti baru untuk ID lama.
- Perubahan entitas wajib memperbarui Data Model dan seluruh diagram ERD terkait.
- Validasi dokumen gagal jika satu UR, FR, BR, atau UF yang dirujuk tidak ditemukan pada dokumen sumber.

## Changelog

### 1.0.0, 23 September 2026

- Membuat baseline keterlacakan untuk 47 user requirement yang disetujui.
