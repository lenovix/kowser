Urutan kerja:
1. M4: Network & Stability (Timeout 15s + Cancelable fetch + Handle redirect + Error page).
2. M3: Form & Search Input (Bisa cari langsung di Google via input query q=).
3. M5: Page Title & Formatting (Set window title dari <title>, bullet list, formatting).
4. M6: Browser Controls (Reload button, Home button).

M3: Form support (wajib dulu)
parseHTML sekarang buang <form>, <input>. Google tanpa search box mati.
Tambah: FormElement{Action, Method, Inputs}, parse form[input[name]], render 1 baris input + tombol, submit rakit GET query via url.Values.Encode(), panggil fetchURL existing.
→ skipped: POST, JS onsubmit. Add when GET search jalan.

M4: Network kokoh
client sekarang tanpa timeout, tanpa cancel, race fetch.
Tambah: http.Client{Timeout:15s}, context.WithCancel batalkan fetch lama, tampil error sebagai PageElement, sync currentURL dari resp.Request.URL.
→ skipped: disk cache, ETag. Add when navigasi terasa lambat.

M5: Render rapi
PageElement sekarang cuma Text/Heading/Link. Judul tab tidak update.
Tambah: ekstrak <title> ke app.Title, bedakan H1-H6, b/strong, ul/li, potong teks 5000 elemen max biar material.List tidak berat.
→ skipped: CSS parser, <img>. Add when teks + link stabil.

M6: UX browser
Belum ada reload, history hilang saat restart.
Tambah: tombol Reload, history.json simpan history, historyIdx, Home ke google.com.
→ skipped: tab, bookmark, download. Add when single-window solid.