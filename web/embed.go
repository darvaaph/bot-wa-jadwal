package web

import "embed"

// Files menyimpan seluruh halaman, partial, dan aset statis frontend di dalam biner.
//
//go:embed index.html app.html superadmin.html pj.html km.html login.html css/* js/* partials/* partials/km/* partials/sa/* partials/pj/* partials/portal/* assets/icons/* assets/illustrations/*
var Files embed.FS
