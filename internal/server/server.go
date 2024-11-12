package server

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Name  string
	Path  string
	IsDir bool
}

func Serve(thumbnails string) {
	mux := http.NewServeMux()
	mux.Handle("/thumb/", http.StripPrefix("/thumb", http.FileServer(http.Dir(thumbnails))))
	mux.HandleFunc("/", fileHandler)
	server := http.Server{
		Addr:           ":9000",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 19,
	}

	fmt.Println("Server starting on port 9000...")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	root, err := filepath.Abs(".")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fullpath, err := filepath.Abs(root + r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(fullpath, root) {
		http.Error(w, "", http.StatusForbidden)
		return
	}

	dir, err := os.Open(fullpath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path, _ := strings.CutPrefix(fullpath, root)
	back := filepath.Dir(path)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<html><head> <meta charset='UTF-8'><title>File Browser</title><style>body { font-family: Arial, sans-serif; } ul { list-style-type: none; padding: 0; } li { margin: 5px 0; } a { text-decoration: none; color: #333; } a:hover { text-decoration: underline; }</style></head><body>")
	fmt.Fprintf(w, "<h1>Browsing: %s</h1><table>", path)
	fmt.Fprintf(w, "<tr><td></td><td><a href=\"%s\">%s %s</a></td></tr>", back, getIcon(true), "..")

	for _, file := range files {
		info := FileInfo{
			Name:  file.Name(),
			Path:  filepath.Join(path, file.Name()),
			IsDir: file.IsDir(),
		}

		if info.IsDir {
			fmt.Fprintf(w, "<tr><td></td><td><a href=\"%s\">📁 %s</a></td></tr>", info.Path, info.Name)
		} else {
			md5 := crypto.MD5.New()
			md5.Write([]byte(fmt.Sprintf("file://%s", filepath.Join(fullpath, info.Name))))
			hash := hex.EncodeToString(md5.Sum(nil))
			fmt.Fprintf(w, "<tr><td><img src=\"/thumb/%s.png\"></td><td><span>%s</span></td></rd>", hash, info.Name)
		}
	}

	fmt.Fprintln(w, "</table></body></html>")
}

func getIcon(isDir bool) string {
	if isDir {
		return "📁"
	}
	return "📄"
}
