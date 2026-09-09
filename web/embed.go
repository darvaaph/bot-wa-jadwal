package web

import "embed"

// Files menyimpan seluruh aset web statis (HTML, CSS, JS) yang disematkan ke dalam biner aplikasi.
//
//go:embed index.html css/* js/*
var Files embed.FS
