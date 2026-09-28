package web

import "embed"

// Files menyimpan seluruh aset web statis (HTML, CSS, JS, partials, aset UI) yang disematkan ke dalam biner aplikasi.
//
//go:embed index.html superadmin.html css/* js/* partials/* partials/km/* partials/sa/* assets/icons/* assets/illustrations/*
var Files embed.FS
