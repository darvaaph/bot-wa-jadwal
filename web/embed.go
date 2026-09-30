package web

import "embed"

// Files menyimpan seluruh halaman, partial, dan aset statis frontend di dalam biner.
//
//go:embed index.html 404.html system-admin.html pj.html km.html login.html invite.html css/* js/* js/vendor/* partials/* partials/common/* partials/km/* partials/system-admin/* partials/pj/* partials/portal/* assets/icons/* assets/illustrations/*
var Files embed.FS
