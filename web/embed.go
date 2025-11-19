// Package web embeds various files for the backend
package web

import (
	"embed"
)

//go:embed build/*
var StaticFS embed.FS
