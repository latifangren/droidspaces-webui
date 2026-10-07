package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

func GetFS() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return http.FS(distFS)
	}
	return http.FS(sub)
}
