# Kowser - Tasks & Development Backlog

Dokumen ini mencatat ringkasan status arsitektur yang sudah selesai serta daftar tugas pengembangan (backlog) yang **belum dilakukan / siap dilanjutkan besok**.

---

## 📌 Ringkasan Status Saat Ini (Sudah Selesai)
Engine peramban telah menyelesaikan migrasi arsitektur dari *flat-list* menjadi pipeline modular engine modern:
- **E1**: DOM Tree Abstraction (`dom.go`, `dom_parser.go`, `QuerySelector`, parent/sibling pointers).
- **E2**: CSS Cascade Engine (`css.go`, `<style>` block parser, Specificity A/B/C, inheritance, computed styles).
- **E3**: Layout Box Engine (`layout.go`, Box Model: margin/padding/content, block flow vertical, margin collapsing, greedy word-wrapping).
- **E4**: Single-Pass Paint Engine (`paint.go`, recursive tree painter ke Gio ops, background, border, image GPU scaling, link hit-testing, dynamic viewport resize).
- **Shell Fitur**: Multi-Tab, Bookmarks Bar, Download Handler, History Stack + persistence (`history.json`, `bookmarks.json`), Cookie Jar.
- **Unit Tests**: 33 unit tests aktif dan seluruhnya lulus (`PASS`).

---

## 🚀 Daftar Tugas yang Belum Didevelop (Development Backlog)

### Prioritas 1: Interaktivitas & Navigasi Viewport (Finishing E4)
- [ ] **Mouse Wheel & Drag Scrolling**:
  - Saat ini `ScrollY` sudah ada di struct `Tab`, tapi scrolling interaktif via mouse wheel (`pointer.Scroll`) dan drag scrollbar belum diikat ke `gtx.Events` di viewport `layoutContent`.
  - Tambahkan gesture handler agar user bisa scroll halaman panjang menggunakan roda mouse atau trackpad.
- [ ] **Keyboard Navigation Shortcuts**:
  - Bind tombol panah `Up`/`Down`, `PageUp`/`PageDown`, `Home`, dan `End` untuk menggeser `tab.ScrollY`.
  - Bind `Ctrl+T` (tab baru), `Ctrl+W` (tutup tab), `Ctrl+R` (reload), `Ctrl+L` (fokus address bar).
- [ ] **Visual Scrollbar Indicator**:
  - Gambar bar scrollbar tipis di sisi kanan viewport yang proporsional terhadap `totalH` vs `viewportH`.

---

### Prioritas 2: Penyempurnaan Form Engine di Layout Tree
- [ ] **In-Tree Form & Input Rendering**:
  - Elemen `<input>` dan `<button>` saat ini masih di-bridge lewat fallback `material.List`.
  - Integrasikan box form (`<form>`, `<input>`, `<button>`) langsung ke dalam `LayoutBox` dan `PaintLayout` agar search box Google ter-render sejajar dalam flow dokumen HTML asli.

---

### Prioritas 3: Fitur Produktivitas Browser
- [ ] **Find in Page (Ctrl+F)**:
  - Baris input pencarian kata kunci di halaman aktif.
  - Highlight kata yang cocok di `LineBox` dan lompat ke posisi kata (`ScrollY`).
- [ ] **Zoom In / Zoom Out (Ctrl + / Ctrl -)**:
  - Faktor skala zoom global (`zoomScale float32 = 1.0`).
  - Kalikan `availWidth`, `fontSize`, dan geometri layout box dengan faktor zoom.
- [ ] **Private / Incognito Mode Tab**:
  - Opsi membuka tab penyamaran dengan cookie jar terpisah dan tanpa riwayat tersimpan ke `history.json`.

---

### Prioritas 4: Engine Lanjutan (E5 - JavaScript Engine)
- [ ] **E5: Evaluasi & Integrasi Goja**:
  - Tambahkan dependensi `github.com/dop251/goja`.
  - Buat jembatan objek global `window` dan `document`.
  - Implementasikan DOM mutation API minimal:
    - `document.getElementById(id)`
    - `document.querySelector(sel)`
    - `element.innerHTML` / `element.textContent`
    - `element.setAttribute` / `element.getAttribute`
    - `document.createElement` + `appendChild`
  - Lifecycle: `Parse DOM` → `Execute <script>` (mutasi DOM) → `ComputeStyles (E2)` → `ComputeLayout (E3)` → `Paint (E4)`.

---

## 🛠️ Catatan Menjalankan Project Besok
Untuk memverifikasi dan menjalankan aplikasi:
```powershell
cd browser
go test -v ./...
go run .
```
