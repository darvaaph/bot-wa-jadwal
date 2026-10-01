-- 012: Normalisasi nomor WhatsApp ke bentuk kanonis +62...
-- Latar: kode lama menyimpan mentah (08... vs +628... vs 62...), sehingga
-- satu nomor bisa jadi dua baris. Kode baru menormalisasi ke +62...,
-- migrasi ini menormalkan baris lama agar login tetap jalan.
-- Idempoten dan aman-konflik: baris yang target kanonisnya sudah dipakai
-- akun lain dilewati (tidak digabung otomatis).

-- 1. Bersihkan pemisah umum (spasi, strip, titik).
UPDATE users SET identity_key = REPLACE(REPLACE(REPLACE(identity_key, ' ', ''), '-', ''), '.', '')
WHERE identity_key LIKE '% %' OR identity_key LIKE '%-%' OR identity_key LIKE '%.%';
UPDATE role_invitations SET invited_identity_key = REPLACE(REPLACE(REPLACE(invited_identity_key, ' ', ''), '-', ''), '.', '')
WHERE invited_identity_key LIKE '% %' OR invited_identity_key LIKE '%-%' OR invited_identity_key LIKE '%.%';

-- 2. Bentuk 08... -> +62... (kasus terbanyak, mis. 085155348696).
UPDATE users SET identity_key = '+62' || SUBSTR(identity_key, 2)
WHERE identity_key LIKE '08%' AND LENGTH(identity_key) >= 10
AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.identity_key = '+62' || SUBSTR(users.identity_key, 2));
UPDATE role_invitations SET invited_identity_key = '+62' || SUBSTR(invited_identity_key, 2)
WHERE invited_identity_key LIKE '08%' AND LENGTH(invited_identity_key) >= 10;

-- 3. Bentuk 62... tanpa plus -> +62....
UPDATE users SET identity_key = '+' || identity_key
WHERE identity_key LIKE '62%' AND identity_key NOT LIKE '+%' AND LENGTH(identity_key) >= 10
AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.identity_key = '+' || users.identity_key);
UPDATE role_invitations SET invited_identity_key = '+' || invited_identity_key
WHERE invited_identity_key LIKE '62%' AND invited_identity_key NOT LIKE '+%' AND LENGTH(invited_identity_key) >= 10;

-- 4. Bentuk 8... tanpa nol (mis. 81234567890) -> +62....
UPDATE users SET identity_key = '+62' || identity_key
WHERE identity_key LIKE '8%' AND identity_key NOT LIKE '08%' AND identity_key NOT LIKE '+%'
AND LENGTH(identity_key) >= 10 AND identity_key NOT GLOB '*[^0-9]*'
AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.identity_key = '+62' || users.identity_key);
UPDATE role_invitations SET invited_identity_key = '+62' || invited_identity_key
WHERE invited_identity_key LIKE '8%' AND invited_identity_key NOT LIKE '08%' AND invited_identity_key NOT LIKE '+%'
AND LENGTH(invited_identity_key) >= 10 AND invited_identity_key NOT GLOB '*[^0-9]*';
