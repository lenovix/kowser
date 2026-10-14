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

### 16. DOM Tree Abstraction (E1)
- Tree struktur DOM (*Node) dengan pointer parent/child/sibling.
- Query engine untuk selector CSS sederhana: `#id`, `.class`, tag, dan `parent child` descendant.
- Flattening DOM ke `PageElement` untuk view saat ini tanpa merusak shell UI browser.
- Mendukung adapter parser dari `golang.org/x/net/html` sebagai input DOM sumber.
- All unit tests lulus untuk tree traversal, querySelector, dan roundtrip parsing HTML ke DOM dan kembali.

### 17. CSS Cascade & Style Computation Engine (E2)
- **Parser CSS**: Membaca blok `<style>` (termasuk penghapusan komentar `/* */`), memecah rule selector `{ ... }`, dan parsing deklarasi `property: value`.
- **Selector Matching Engine**: Mencocokkan selector `tag`, `.class`, `#id`, dan compound (`div.cls#id`) serta descendant combinator (`parent child`) terhadap node DOM.
- **Specificity Calculation**: Perhitungan bobot selector `(ID, Class, Tag)` dan sorting stabil berdasarkan specificity lalu urutan dokumen.
- **Cascade Resolution**: Penerapan berurutan dari specificity terendah ke tertinggi, dengan atribut inline `style="..."` sebagai override prioritas tertinggi.
- **Inheritance**: Properti yang dapat diwariskan (`color`, `font-size`) diturunkan dari parent ke child secara rekursif.
- **ComputedStyle**: Struktur nilai visual final per node: `Display`, `Color`, `BackgroundColor`, `FontSize`, `IsBold`, `IsItalic`, `Width`, `Height`, `Margin`, `Padding`.
- **Default User-Agent Styles**: Style bawaan per tag (`h1`-`h6` bold + margin, `a` biru, `b/strong` bold, `em/i` italic, `span/a` inline).
- **Integrasi Pipeline**: `ComputeStyles(domRoot)` dijalankan otomatis setelah parsing HTML; `flattenDOMToPageElements` menggunakan `n.Computed` sebagai sumber styling utama.

### 18. Layout Box Engine (E3)
- **Layout Tree Builder**: `BuildLayoutTree` mengonversi DOM tree menjadi box tree geometri, dengan dukungan `display: none` (dibuang dari tree) dan klasifikasi `Block` / `Inline`.
- **CSS Box Model**: Perhitungan lengkap empat lapis box: `margin`, `padding`, `content (width/height)`, disertai helper geometri `ContentX/Y`, dan `PaddingBox`/`MarginBox`.
- **Block Flow Layout**: Anak-anak block disusun vertikal, lebar blok otomatis mengisi lebar containing block (kecuali `width` eksplisit).
- **Vertical Margin Collapsing**: Gap antar sibling block memakai `max(prevMarginBottom, nextMarginTop)` sesuai standar CSS.
- **Inline Formatting Context & Word Wrapping**: Teks dipecah menjadi kata-kata dan dibungkus otomatis pada batas kata (`word boundary`) sesuai lebar container, menghasilkan `LineBox` per baris dengan `Y` dan `Height` yang akurat.
- **Whitespace Handling**: Text node yang hanya berisi whitespace langsung di bawah elemen block tidak menghasilkan layout box (mencegah margin collapsing rusak dan gap palsu).
- **Text Measurement**: Estimasi lebar karakter & tinggi baris proporsional terhadap `font-size` (dapat digantikan oleh font metrics Gio yang sesungguhnya pada E4).
- **Integrasi Pipeline**: `BuildLayoutTree` + `ComputeLayout` dijalankan otomatis setelah CSS cascade; hasilnya disimpan sebagai `tab.LayoutRoot` untuk dikonsumsi oleh Paint Phase (E4).

### 19. Paint Phase — Single-Pass Rendering Engine (E4)
- **paint.go**: Modul pemerataan `PaintLayout` dan `PaintContext` untuk merender `LayoutBox` tree menjadi Gio paint ops dalam satu kali travers, mengikuti arsitektur "paint after layout" yang benar (tidak lagi flat-list).
- **Painting Order**: Background padding box → uniform box border → text glyphs (via `material.Label`) → images (via `op/paint` GPU image ops) → link decorations, dalam urutan rekursif blok.
- **Word-Line Rendering**: Setiap `LineBox` dilukis dengan `material.Label` per kata kecil dengan offset absolut dan pewarnaan link yang tepat (biru Google / computed style).
- **Responsive Re-Layout**: `ComputeLayout(root, viewportWidth)` dijalankan ulang tiap `FrameEvent` di `layoutContent` berdasarkan batas viewport Gio (`gtx.Constraints.Max.X`), sehingga dokumen menyesuaikan lebar jendela secara dinamis.
- **Scrollable Viewport**: Primitive scroll vertikal per-`Tab` (`ScrollY`, `MaxScrollY`); scroll position diterapkan sebagai offset global `absY - ScrollY` pada semua node paint.
- **Link Hit-Testing & Routing**: `PaintContext.Links map[*Node]*widget.Clickable` dialokasi stabil per-`Tab`; hit rect dihitung dari geometri `LineBox`, dan klik terdeteksi via `Clickable.Clicked(gtx)` → ekstraksi `href` yang langsung di-resolve ke `fetchURL`.
- **Image Pipeline Integration**: `<img>` diambil dan di-decode secara async, di-cache di `imgCache`, dan diskalakan secara GPU-accelerated di `PaintImageBox` dengan letterboxing proporsional.
- **Fallback Path**: Status awal tanpa layout tree tetap memakai `material.List` legacy view untuk halaman error/loading, sehingga shell UI tetap kompatibel selama transisi.
- **Tests**: `paint_test.go` — validasi emisi paint yang benar, alokasi clickable per link, dan perhitungan total height tanpa click palsu.

---

## Roadmap Fitur Berikutnya

- [ ] **E5: JavaScript Engine** (Integrasi Goja, bridge DOM API) — ditunda sampai E1–E4 stabil.

**Catatan penting**: Fondasi engine telah mencapai **E4 (Paint Phase — Single-Pass Rendering Engine)**. Dokumen statis sekarang benar-benar "paint after geometry": X/Y/Width/Height akurat di memori, word wrapping, margin collapsing, dan pewarnaan CSS cascade semua berakhir di single-pass paint ke GPU Gio. Tahap berikutnya adalah **E5 (JavaScript Engine / DOM Bindings)** yang merupakan lonjakan ambisius terbesar dan hanya akan dimulai setelah E1–E4 terbukti stabil di penggunaan nyata.