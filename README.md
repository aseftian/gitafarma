# Aplikasi Apotek Gita Farma

Aplikasi PWA satu sumber referensi harga obat untuk karyawan Apotek Gita Farma, tersinkronisasi dengan Google Sheet.

## Struktur Monorepo

```text
.
├── backend/        # Go REST API + Sync Scheduler + SQLite + Web Push (Clean Architecture)
├── web/            # React + TypeScript + Vite + Tailwind + shadcn/ui + PWA
├── deploy/         # Konfigurasi reverse proxy HTTPS (Caddy)
├── docs/           # Kebutuhan & ADR (Architecture Decision Records)
├── Makefile
├── docker-compose.yml
└── docker-compose.prod.yml
```

## Prasyarat & Setup Google Cloud

1. **Google Spreadsheet (1 spreadsheet, 2 tab):**
   - Tab `medicines`: header `id`, `name`, `barcode`, `sku`, `price`, `uom`, `alt_uom`, `updated_at`
   - Tab `admins`: header `email`, `name`, `role`, `active` (isi `TRUE` untuk admin aktif)
2. **Google Workload Identity Federation (WIF — Tanpa Service Account JSON Key):**
   - Aktifkan **Google Sheets API**, **IAM Service Account Credentials API**, dan **Security Token Service API** di Google Cloud Console.
   - Buat Service Account (tanpa membuat JSON key), lalu bagikan (Share) Spreadsheet ke email Service Account tersebut sebagai **Editor**.
   - Buat **Workload Identity Pool** dan **OIDC Provider** di GCP IAM, lalu berikan role `roles/iam.workloadIdentityUser` pada Service Account untuk principal OIDC Anda.
   - Generate file konfigurasi WIF (`"type": "external_account"` — tidak berisi private key) untuk VPS/Docker:
     ```bash
     mkdir -p credentials
     gcloud iam workload-identity-pools create-cred-config \
       projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID \
       --service-account=SERVICE_ACCOUNT_EMAIL \
       --credential-source-file=/app/credentials/oidc-token \
       --output-file=./credentials/wif-config.json
     ```
     *(Simpan/perbarui file OIDC token di `./credentials/oidc-token` pada host VPS; folder `./credentials` otomatis di-mount read-only ke `/app/credentials` di container Docker. Jika OIDC token diambil dari endpoint HTTP lokal, ganti `--credential-source-file` dengan `--credential-source-url`.)*
   - **Alternatif untuk Development Lokal (Tanpa File Key):**
     Kosongkan `GOOGLE_APPLICATION_CREDENTIALS=` di `.env`, lalu jalankan:
     ```bash
     gcloud auth application-default login --impersonate-service-account=SERVICE_ACCOUNT_EMAIL
     ```
3. **Google OAuth 2.0 Client ID (Web):**
   - Buat OAuth 2.0 Client ID bertipe **Web application** di Google Cloud Console.
   - Tambahkan **Authorized JavaScript origins** (mis. `http://localhost:5173` untuk dev dan `https://harga.domainanda.com` untuk produksi).
   - Isi Client ID ke `GOOGLE_OAUTH_CLIENT_ID` dan `VITE_GOOGLE_CLIENT_ID` di `.env`.
4. **VAPID Keys (Web Push):**
   ```bash
   make vapid-keys
   ```
   Salin `VAPID_PUBLIC_KEY` dan `VAPID_PRIVATE_KEY` ke `.env`.

## Menjalankan Secara Lokal

1. Salin `.env.example` menjadi `.env` dan isi variabel yang diperlukan.
2. Jalankan backend & web di dua terminal terpisah:
   ```bash
   make dev-backend
   make dev-web
   ```

## Deployment Produksi (Docker + HTTPS Otomatis)

Kamera (barcode & OCR), PWA install, dan Web Push membutuhkan HTTPS. Gunakan Caddy reverse proxy dengan TLS otomatis:

```bash
# Pastikan APP_DOMAIN, SESSION_COOKIE_SECURE=true, APP_ENV=production sudah diatur di .env
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d
```

## Pengujian

```bash
make test
make lint
```

## Dokumentasi

- [Requirements](docs/Requirements.md)
- [ADR 0001: Storage Engine](docs/adr/0001-storage.md)
