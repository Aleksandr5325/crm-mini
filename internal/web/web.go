// Package web раздаёт веб-интерфейс, встроенный в бинарник приложения.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// static — файлы интерфейса. Директива go:embed встраивает папку static в бинарник при сборке,
// поэтому для запуска приложения не нужны никакие файлы рядом с ним.
//
//go:embed static
var static embed.FS

// Register добавляет в mux раздачу интерфейса на корневом адресе.
// Маршруты /api/v1/... конкретнее, чем "GET /", поэтому их не перекрывает.
func Register(mux *http.ServeMux) {
	files, err := fs.Sub(static, "static")
	if err != nil {
		// Папка static встроена при сборке: если её нет, программа собрана неправильно.
		panic(err)
	}

	mux.Handle("GET /", http.FileServerFS(files))
}
