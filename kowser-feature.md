# Kowser - Daftar Fitur

Daftar fitur browser Kowser yang sudah di-develop dan yang masih dalam antrean (sesuai tasks.md).

## Fitur yang Sudah Selesai (Completed)

### 1. Window & UI Dasar (Gio UI)
- Desktop window murni Go via Gio UI (`gioui.org`), tanpa dependensi CGO.
- Layout responsif: Tab Bar, Top Bar (Address bar & Navigasi), Bookmark Bar, Status Bar, Content Viewport.

### 2. Address Bar & Navigasi Manual
- Input teks URL dengan event enter (`SubmitEvent`) dan tombol `Go`.
- Normalisasi URL otomatis: prefixing protokol `https://` jika pengguna mengetik tanpa protokol (misal: `google.com` -> `https://google.com`).

### 3. Hyperlink Click Navigation
- Ekstraksi tag `<a>` HTML dan atribut `href`.
- Rendering link dengan warna khusus (biru) dan clickable.
- URL resolution otomatis via `url.ResolveReference` (mendukung relative path seperti `/search` ke domain aktif).

### 4. History (Back & Forward)
- Manajemen tumpukan riwayat penelusuran (`history []string`, `historyIdx int`) terisolasi per tab.
- Tombol navigasi `<` (Back) dan `>` (Forward) yang sinkron dengan URL aktif dan address bar.

### 5. Tipografi Dasar Konten
- Pembedaan elemen teks: Heading (`<h1>` - `<h6>`) dan teks paragraf/konten biasa (`<p>`, `<div>`, `<li>`, `<span>`).
- Filter tag non-konten (`<script>`, `<style>`, `<noscript>`, `<svg>`).

### 6. State & Cookie Management
- HTTP client menggunakan in-memory cookie jar (`net/http/cookiejar`) untuk mempertahankan cookie sesi navigasi.

### 7. Network & Stability (M4)
- HTTP client dengan timeout 15 detik (`http.Client{Timeout: 15 * time.Second}`).
- Cancelable fetch via `context.WithTimeout` + `context.CancelFunc` per tab untuk mencegah race condition saat navigasi cepat.
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

### 11. Image Rendering Engine (M8)
- Deteksi dan ekstraksi tag `<img>` beserta atribut `src` dan `alt`.
- Resolusi link gambar relatif ke URL absolut otomatis.
- Background asynchronous downloading (`fetchImageAsync`) menggunakan context timeout 10 detik agar antarmuka tidak freeze.
- Decoding format raster standar Go (`image/png`, `image/jpeg`, `image/gif`).
- In-memory image cache (`map[string]image.Image`) untuk menghindari network call berulang.
- Layout rendering gambar via `widget.Image` + `paint.NewImageOp` dengan batasan ukuran responsif (max bounding box).
- Placeholder atau teks alternatif (alt text) saat gambar sedang diunduh atau gagal dimuat.

### 12. Bookmarks & Quick Access Bar (M10)
- Tombol bookmark dinamis (`★` / `☆`) di bar atas yang mendeteksi status bookmark URL aktif secara otomatis.
- Toggle tambah/hapus bookmark dengan sekali klik.
- Persistent bookmark disimpan dan dimuat dari file `bookmarks.json`.
- Bookmark Bar horizontal yang menampilkan daftar link favorit tepat di bawah address bar untuk navigasi langsung.

### 13. Multi-Tab Support (M9)
- Komponen Tab Bar horizontal di bagian paling atas aplikasi dengan tombol tambah tab (`+`).
- Setiap tab memiliki state terisolasi penuh: URL aktif, riwayat navigasi (history stack & index), elemen halaman (`PageElements`), status loading/error, dan context cancellation.
- Kemampuan berpindah antar tab (klik judul tab) secara instan.
- Judul tab dinamis yang mencerminkan judul halaman atau URL aktif (dengan pemotongan panjang teks agar rapi).
- Tombol tutup tab individual (`x`).
- Perlindungan tab terakhir: jika tab terakhir ditutup, browser tidak crash melainkan otomatis me-reset tab menjadi "New Tab" ke `google.com`.

### 14. File Download Handler (M11)
- Deteksi tipe konten unduhan otomatis via `Content-Type` header biner (`application/pdf`, `zip`, `octet-stream`, `tar`, `gzip`, audio, video) dan header `Content-Disposition: attachment`.
- Pengalihan otomatis dari pipeline HTML parser langsung ke stream writer biner lokal.
- Deteksi nama berkas via parameter `filename` di Content-Disposition atau fallback ke path URL terakhir.
- Penyimpanan berkas langsung ke direktori bawaan sistem `Downloads` pengguna (`os.UserHomeDir()/Downloads`).
- Tampilan notifikasi status download dan halaman ringkasan download berhasil (dengan lokasi berkas dan ukuran byte).

### 15. Inline CSS Styling Engine (M7)
- Parser atribut HTML `style="..."` pada tag heading (`h1-h6`), link (`a`), bullet list (`li`), teks tebal (`b/strong`), dan paragraf (`p/div/span`).
- Dukungan warna font `color`:
  - Format kode heksadesimal (`#RRGGBB` dan `#RGB`).
  - Named colors standar CSS (`red`, `green`, `blue`, `black`, `gray`, `orange`, `purple`, `yellow`, dll).
- Dukungan ukuran font `font-size`:
  - Unit pixel absolut (`12px`, `16px`, `24px`, dll).
  - Skala tipografi semantik (`small`, `medium`, `large`, `x-large`, `xx-large`).
- Dukungan properti `background-color` / `background`.
- Render tipografi responsif Gio dengan injeksi `StyleProps` dinamis per-elemen pada layar.

---

## Roadmap Fitur Berikutnya (Sesuai `tasks.md`)

- [ ] **M12: Zoom Controls & Text Resizing** (Zoom in/out skala tampilan konten seluruh viewport).
- [ ] **M13: Find in Page (Ctrl+F)** (Pencarian kata kunci dalam konten halaman aktif dan highlight posisi).
- [ ] **M14: Private / Incognito Tab Mode** (Mode penjelajahan tanpa penyimpanan cookies dan history).
