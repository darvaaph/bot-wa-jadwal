# 🚀 Panduan Deployment Produksi — Bot Jadwal v3.0

Panduan langkah demi langkah untuk mendistribusikan biner tunggal (*single binary*) Bot WhatsApp Jadwal Kuliah & Web Dashboard ke server Linux (Ubuntu/Debian) di Microsoft Azure VM.

---

## 📋 1. Prasyarat Server & Lingkungan

1. **Spesifikasi Server Minimum:**
   - 1 vCPU / 1 GB RAM (misal: Azure B1s / B1ms).
   - Ubuntu 22.04 LTS / 24.04 LTS atau Debian 12.
   - Storage SSD minimal 10 GB.
2. **Perangkat Lunak Pendukung:**
   - Reverse Proxy: **Nginx** atau **Caddy** (dengan sertifikat SSL/TLS gratis via Let's Encrypt).
   - Utility: `curl`, `openssl`, `tar`, `systemd`.
3. **Domain / Subdomain Resmi Kampus:**
   - Domain dengan DNS A Record yang mengarah ke IP publik Azure VM Anda (misal: `jadwal.jurusan-anda.ac.id`).
4. **Nomor WhatsApp Resmi Bot:**
   - Nomor WhatsApp aktif pada ponsel fisik pengurus/admin untuk memindai QR Code saat pertama kali inisialisasi.

---

## 🛠️ 2. Kompilasi Biner (Bisa Dilakukan di Lokal / CI)

Karena proyek ini menggunakan fitur kompilasi silang (*cross-compile*) Go tanpa build-step npm:

```bash
# Kompilasi untuk server Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/bot ./cmd/bot
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/seed ./cmd/seed
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/provision-admin ./cmd/provision-admin
```

Unggah folder proyek beserta biner yang sudah dikompilasi ke server:
```bash
rsync -avz --exclude 'storage/*.db' --exclude '.git' . user@ip-server-azure:/opt/bot-jadwal/
```

---

## 👤 3. Konfigurasi Pengguna & Direktori Server

Jalankan perintah ini di server Linux untuk mengamankan proses (menghindari menjalankan bot sebagai `root`):

```bash
# 1. Buat pengguna sistem khusus tanpa shell interaktif
sudo useradd -r -s /bin/false -d /opt/bot-jadwal botuser

# 2. Buat folder penyimpanan database & log
sudo mkdir -p /opt/bot-jadwal/storage /opt/bot-jadwal/bin
sudo chown -R botuser:botuser /opt/bot-jadwal
sudo chmod -R 750 /opt/bot-jadwal
sudo chmod -R 770 /opt/bot-jadwal/storage
```

---

## 🔐 4. Konfigurasi Berkas `.env`

Salin template konfigurasi ke `/opt/bot-jadwal/.env`:

```bash
cd /opt/bot-jadwal
cp .env.example .env
chmod 600 .env
chown botuser:botuser .env
```

Sunting file `.env`:
```ini
BOT_JADWAL_ENV=production
PORT=8080
STORAGE_DIR=/opt/bot-jadwal/storage

# Hasilkan kunci 32-byte acak: openssl rand -hex 32
BOT_JADWAL_AUTH_HASH_KEY=paste_hasil_openssl_di_sini_minimal_32_karakter

BOT_JADWAL_SECURE_COOKIES=true
BOT_JADWAL_PUBLIC_BASE_URL=https://jadwal.jurusan-anda.ac.id
BOT_JADWAL_ALLOWED_ORIGINS=https://jadwal.jurusan-anda.ac.id
BOT_JADWAL_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
```

---

## 🗄️ 5. Inisialisasi Database & Pembuatan Admin Awal

Sebelum menjalankan layanan bot sebagai daemon di background:

### A. Seed Master Data 19 Kelas ke Database v1
```bash
sudo -u botuser /opt/bot-jadwal/bin/seed \
  -manifest /opt/bot-jadwal/data/migration_manifest.json \
  -db /opt/bot-jadwal/storage/bot_v1.db \
  -legacy-db /opt/bot-jadwal/storage/tugas.db
```

### B. Buat Akun System Administrator Pertama
```bash
export BOT_JADWAL_AUTH_HASH_KEY="kunci_rahasia_dari_file_env"
export BOT_JADWAL_ADMIN_PASSWORD="PasswordSangatKuatDanRahasia123!"

sudo -u botuser -E /opt/bot-jadwal/bin/provision-admin \
  -identity "+6281234567890" \
  -display-name "System Administrator" \
  -database /opt/bot-jadwal/storage/bot_v1.db

# Bersihkan password dari riwayat terminal
unset BOT_JADWAL_ADMIN_PASSWORD
```

---

## 🤖 6. Pengaturan Service Systemd (`bot-jadwal.service`)

Buat berkas unit systemd di `/etc/systemd/system/bot-jadwal.service`:

```ini
[Unit]
Description=Bot WhatsApp Jadwal Kuliah & Web Dashboard
After=network.target

[Service]
Type=simple
User=botuser
Group=botuser
WorkingDirectory=/opt/bot-jadwal
EnvironmentFile=/opt/bot-jadwal/.env
ExecStart=/opt/bot-jadwal/bin/bot -session /opt/bot-jadwal/storage/sesi_bot.db
Restart=always
RestartSec=10
LimitNOFILE=65535

# Proteksi keamanan proses Linux
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Aktifkan dan jalankan unit:
```bash
sudo systemctl daemon-reload
sudo systemctl enable bot-jadwal
```

> **Catatan Pemindaian QR WhatsApp Pertama Kali:**  
> Untuk scan QR code pertama kali, jalankan biner secara interaktif di terminal terlebih dahulu:
> ```bash
> sudo -u botuser /opt/bot-jadwal/bin/bot -session /opt/bot-jadwal/storage/sesi_bot.db
> ```
> Pindai kode QR yang muncul di terminal menggunakan aplikasi WhatsApp di HP bot Anda. Setelah terhubung (`Login sukses!`), hentikan proses (`Ctrl+C`) lalu start via systemd:
> ```bash
> sudo systemctl start bot-jadwal
> sudo systemctl status bot-jadwal
> ```

---

## 🌐 7. Konfigurasi Reverse Proxy Nginx

Buat konfigurasi situs di `/etc/nginx/sites-available/bot-jadwal`:

```nginx
server {
    listen 80;
    server_name jadwal.jurusan-anda.ac.id;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name jadwal.jurusan-anda.ac.id;

    # SSL Certificate (dikelola oleh Certbot)
    ssl_certificate /etc/letsencrypt/live/jadwal.jurusan-anda.ac.id/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/jadwal.jurusan-anda.ac.id/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    client_max_body_size 10M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        # Header keamanan dan penerusan IP asli
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (jika dibutuhkan)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_connect_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

Aktifkan konfigurasi:
```bash
sudo ln -s /etc/nginx/sites-available/bot-jadwal /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 🩺 8. Verifikasi Kesiapan Operasional (Health & Ready Check)

Setelah server menyala di balik domain resmi:

1. **Uji Endpoint Liveness & Readiness:**
   ```bash
   curl -I https://jadwal.jurusan-anda.ac.id/api/health
   # Respons: HTTP/2 200 OK

   curl -I https://jadwal.jurusan-anda.ac.id/api/ready
   # Respons: HTTP/2 200 OK (menandakan DB v1 aktif, migrasi terpasang, dan WhatsApp online)
   ```
2. **Login System Admin:**
   - Akses `https://jadwal.jurusan-anda.ac.id/login.html`
   - Masukkan identitas `+6281234567890` dan password admin Anda.
3. **Hubungkan Grup WhatsApp ke Kelas:**
   - Buka menu **System Admin** -> **WhatsApp Channels**.
   - Cari JID Grup WhatsApp kelas mahasiswa, lalu tautkan ke kelas sasaran (misal: `D4-TI-2024-A`).
4. **Verifikasi Interaksi Bot:**
   - Di grup WhatsApp yang sudah ditautkan, ketik: `!jadwal` dan `!portal`.
   - Pastikan bot membalas dengan ringkasan jadwal yang benar dan tautan portal resmi berdomain HTTPS.
