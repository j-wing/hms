package hms

import (
	//"fmt"
	"io/ioutil"
	"net/http"
)

var (
	indexTmpl []byte
	routes    map[string]string
)

func init() {
	content, err := ioutil.ReadFile("index.html")
	if err != nil {
		panic(err)

	}
	indexTmpl = content

	http.HandleFunc("/", ShortenerHandler)
	http.HandleFunc("/wat", WatRedirect)
}

func redirectHandler(loc string, w http.ResponseWriter) {
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusFound)
}

func FBRedirect(w http.ResponseWriter, r *http.Request) {
	redirectHandler("https://www.facebook.com/messages/conversation-807942749260663", w)
}

func WatRedirect(w http.ResponseWriter, r *http.Request) {
	redirectHandler("https://www.google.com/search?site=&tbm=isch&source=hp&biw=1920&bih=969&q=wat&oq=wat&gs_l=img.12...0.0.0.2619.0.0.0.0.0.0.0.0..0.0....0...1ac..64.img..0.0.0.Zioxed2_GrU", w)
}

func createShortenedURL(r *http.Request) (string, string) {
	path := r.FormValue("path")
	target := r.FormValue("target")

	if path == "" || target == "" {
		return "", "empty path or target"
	} else {
		routes[path] = target
		return path, ""
	}
}

func writeNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("No match!"))
}

func ShortenerHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	if routes == nil {
		routes = make(map[string]string)
		writeNotFound(w)
	}

	if reqPath == "/" {
		if r.Method == "GET" {
			w.Write(indexTmpl)
		} else if r.Method == "POST" {
			resURL, err := createShortenedURL(r)
			if err != "" {
				w.Write([]byte(err))
			} else {
				w.Header().Set("Location", resURL)
				w.WriteHeader(http.StatusFound)
			}
		}
	} else {

		match := routes[reqPath]
		if match != "" {
			w.Header().Set("Location", match)
			w.WriteHeader(http.StatusFound)
		} else {
			writeNotFound(w)
		}
	}
}
