package web

import "embed"

// Files menyimpan seluruh halaman, partial, dan aset statis frontend di dalam biner.
//
//go:embed index.html 404.html system-admin.html pj.html km.html login.html invite.html manifest.webmanifest manifest.json sw.js favicon.ico css/* js/* js/vendor/* partials/* partials/common/* partials/km/* partials/system-admin/* partials/pj/* partials/portal/* assets/icons/* assets/illustrations/* assets/fonts/*
var Files embed.FS
