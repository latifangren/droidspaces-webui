package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var DistFS embed.FS

func GetFS() http.FileSystem {
	sub, err := fs.Sub(DistFS, "dist")
	if err != nil {
		return http.FS(DistFS)
	}
	return http.FS(sub)
}
