package prompt

import "embed"

//go:embed all:defaults
var embeddedFS embed.FS

const defaultsDir = "defaults"
