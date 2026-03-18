# Subscription Voice Usage Design

## 1) PostgreSQL Schema

### `app_subscriptions`
- Menyimpan kuota plan bulanan aktif per user.
- Field utama:
  - `plan_id` (`starter`, `pro`, `elite`)
  - `total_voice_minutes`, `used_voice_minutes`
  - `total_sessions_limit`, `used_sessions`
  - `period_start`, `period_end`
- Period reset bulanan menggunakan rentang `period_start` → `period_end`.

### `app_usage_tracking`
- Ledger/idempotency log penggunaan.
- Field utama:
  - `usage_type` (`voice_minutes`, `session_count`)
  - `consumed_minutes`, `consumed_sessions`
  - `session_id` (nullable)
  - `period_start`, `period_end`
  - `metadata` (JSONB)
- Unique constraint:
  - `user_id + session_id + usage_type + period_start` (saat `session_id` ada)
  - mencegah double charge pada retry/reconnect.

## 3) Redis Usage (Optional, Recommended)

### Cache strategy
- Key: `subscription:state:{user_id}`
- Isi: snapshot state subscription aktif (JSON)
- TTL: 60 detik

### Read path
1. Cek Redis.
2. Jika miss/expired, baca PostgreSQL.
3. Simpan ulang ke Redis.

### Write path & sync
- Setelah event mutasi (`ConsumeSession`, `CommitVoiceUsage`), backend update PostgreSQL dulu.
- Setelah commit sukses, refresh cache Redis dengan state terbaru.
- Jika Redis gagal, request tetap sukses (degrade gracefully).

### Rate limiting (opsional lanjutan)
- Per user endpoint `voice/agent/session`: 5 request / 30 detik.
- Per user endpoint `voice/usage/commit`: 10 request / menit.
- Bisa pakai key Redis berbasis sliding-window/token-bucket.

## 5) Cost Control Strategy

### Hard gate sebelum voice start
- Endpoint `POST /api/voice/agent/session`:
  - cek `remaining_voice_minutes`.
  - jika 0, reject (`402` / message quota habis).

### Auto stop saat kuota habis
- Backend kirim `allowed_call_seconds`.
- Frontend memotong durasi call maksimal dengan nilai ini.
- Saat timer mencapai batas, session diakhiri otomatis.

### Warning threshold < 10%
- Backend kirim `warning_threshold_reached` jika sisa <=10%.
- Frontend menampilkan warning dan tetap lanjut sampai limit.

### Fallback ke text mode
- Jika start voice ditolak karena kuota habis, frontend arahkan user kembali ke halaman practice (text mode tetap tersedia).

### Idempotent usage commit
- Endpoint `POST /api/voice/usage/commit` memakai constraint unik per session.
- Retry tidak menggandakan charge usage.

### Fixed plan policy
- Kuota voice mengikuti batas plan bulanan (`starter`, `pro`, `elite`) tanpa mekanisme top-up.
- Saat kuota habis, user tetap dapat lanjut latihan text mode sampai periode berikutnya.
