package web

import "embed"

// Files menyimpan seluruh aset web statis (HTML, CSS, JS, partials, aset UI) yang disematkan ke dalam biner aplikasi.
//
//go:embed index.html css/* js/* partials/* assets/icons/* assets/illustrations/*
var Files embed.FS
