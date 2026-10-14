# Kowser - Daftar Fitur

Daftar fitur browser Kowser yang sudah di-develop dan yang masih dalam antrean (sesuai tasks.md).

## Fitur yang Sudah Selesai (Completed)

### 1. Window & UI Dasar (Gio UI)
- Desktop window murni Go via Gio UI (`gioui.org`), tanpa dependensi CGO.
- Layout responsif: Top Bar (Address bar & Navigasi), Status Bar, Content Viewport.

### 2. Address Bar & Navigasi Manual
- Input teks URL dengan event enter (`SubmitEvent`) dan tombol `Go`.
- Normalisasi URL otomatis: prefixing protokol `https://` jika pengguna mengetik tanpa protokol (misal: `google.com` -> `https://google.com`).

### 3. Hyperlink Click Navigation
- Ekstraksi tag `<a>` HTML dan atribut `href`.
- Rendering link dengan warna khusus (biru) dan clickable.
- URL resolution otomatis via `url.ResolveReference` (mendukung relative path seperti `/search` ke domain aktif).

### 4. History (Back & Forward)
- Manajemen tumpukan riwayat penelusuran (`history []string`, `historyIdx int`).
- Tombol navigasi `<` (Back) dan `>` (Forward) yang sinkron dengan URL aktif dan address bar.

### 5. Tipografi Dasar Konten
- Pembedaan elemen teks: Heading (`<h1>` - `<h6>`) dan teks paragraf/konten biasa (`<p>`, `<div>`, `<li>`, `<span>`).
- Filter tag non-konten (`<script>`, `<style>`, `<noscript>`, `<svg>`).

### 6. State & Cookie Management
- HTTP client menggunakan in-memory cookie jar (`net/http/cookiejar`) untuk mempertahankan cookie sesi navigasi.

### 7. Network & Stability (M4)
- HTTP client dengan timeout 15 detik (`http.Client{Timeout: 15 * time.Second}`).
- Cancelable fetch via `context.WithTimeout` + `context.CancelFunc` untuk mencegah race condition saat navigasi cepat.
- Error page element (`TypeError`) yang menampilkan pesan error di viewport jika fetch gagal.
- Sinkronisasi `currentURL` dari `resp.Request.URL` (handle redirect otomatis).

### 8. Form & Search Input (M3)
- Parsing tag `<form>` dengan atribut `action` dan `method`.
- Render form sebagai input field + tombol "Search" di viewport.
- Submit form GET: query string dirakit via `url.Values.Encode()` dan parameter `q` di-set dari input field.
- Integrasi dengan `fetchURL` existing untuk navigasi otomatis saat form disubmit.

### 9. Render Rapi (M5)
- Ekstraksi `<title>` dari HTML dan set ke window title via `app.Title(title)`.
- Pembedaan level heading: `H4` untuk `<h1>`, `H5` untuk `<h2>`, `H6` untuk `<h3>`-`<h6>`.
- Bullet list: tag `<li>` dirender dengan prefix `•`.
- Bold text: tag `<strong>` dan `<b>` diekstrak sebagai elemen teks terpisah.
- Limit 5000 elemen maksimal untuk mencegah `material.List` overload.

### 10. UX Browser (M6)
- Tombol Reload (`R`) untuk memuat ulang halaman aktif.
- Tombol Home (`H`) untuk kembali ke `https://www.google.com`.
- Persistent history: simpan `history` dan `historyIdx` ke `history.json` saat navigasi dan saat aplikasi ditutup.
- Load history dari `history.json` saat aplikasi dimulai (restore sesi terakhir).

---

## Roadmap Fitur Berikutnya (Sesuai `tasks.md`)

- [ ] **M7: CSS Parser Dasar** (Parsing stylesheet inline `<style>` dan attribute `style`, apply warna/font-size dasar).
- [ ] **M8: Image Rendering** (Ekstraksi `<img>` dan render thumbnail via Gio image widget).
- [ ] **M9: Tab Support** (Multi-tab browsing dengan state terpisah per tab).
- [ ] **M10: Bookmark** (Simpan URL favorit ke `bookmarks.json`, UI untuk add/remove bookmark).
