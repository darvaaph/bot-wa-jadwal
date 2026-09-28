package web

import "embed"

//go:embed index.html app.html css js
var Files embed.FS
