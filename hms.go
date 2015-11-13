package hms

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
)

var (
	indexTmpl = template.Must(template.ParseFiles("index.html"))
	routes    map[string]string
)

const letterRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type URLMatch struct {
	Path      string
	TargetURL string
	Creator   string
	Created   time.Time
}

type IndexTemplateParams struct {
	Path       string
	TargetURL  string
	Message    string
	CreatedURL string
}

func URLMatchKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "URLMatch", "default_urlmatch", 0, nil)

}

func createRandomPath(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}

func init() {
	http.HandleFunc("/", ShortenerHandler)
}

func redirectHandler(loc string, w http.ResponseWriter) {
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusFound)
}

func FBRedirect(w http.ResponseWriter, r *http.Request) {
	redirectHandler("https://www.facebook.com/messages/conversation-807942749260663", w)
}

func isValidPath(path string) bool {
	return true
}

func isValidTargetURL(target string) bool {
	return !strings.Contains(target, "hms.space")
}

func createShortenedURL(r *http.Request) (string, string) {
	path := r.FormValue("path")
	target := r.FormValue("target")

	if target == "" {
		return "", "empty target"
	} else {
		if path == "" {
			path = createRandomPath(4)
		} else if !isValidPath(path) {
			return "", "invalid path"
		} else if !isValidTargetURL(target) {
			return "", "invalid target url"
		}
		c := appengine.NewContext(r)
		u := URLMatch{
			Path:      path,
			TargetURL: target,
			Creator:   "",
			Created:   time.Now(),
		}

		key := datastore.NewIncompleteKey(c, "URLMatch", URLMatchKey(c))
		_, err := datastore.Put(c, key, &u)
		if err != nil {
			return "", "ERROR"
		}
		return path, ""
	}
}

func writeNotFound(w http.ResponseWriter, path string) {
	indexTmpl.Execute(w, IndexTemplateParams{
		Path:    path[1:],
		Message: "No entry for that path. Add one?",
	})
}

func ShortenerHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path

	if reqPath == "/" {
		if r.Method == "GET" {
			indexTmpl.Execute(w, IndexTemplateParams{
				Path:      r.FormValue("path"),
				TargetURL: r.FormValue("targetURL"),
			})
		} else if r.Method == "POST" {
			resURL, err := createShortenedURL(r)
			if err != "" {
				http.Error(w, err, http.StatusInternalServerError)
			} else {
				fullURL := fmt.Sprintf("%v/%v", r.Host, resURL)
				indexTmpl.Execute(w, IndexTemplateParams{
					CreatedURL: fullURL,
				})
			}
		}
	} else {
		c := appengine.NewContext(r)
		match := make([]URLMatch, 0, 1)
		_, err := datastore.NewQuery("URLMatch").Filter("Path =", reqPath[1:]).Limit(1).GetAll(c, &match)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

		}

		if len(match) != 0 {
			w.Header().Set("Location", match[0].TargetURL)
			w.WriteHeader(http.StatusFound)
		} else {
			writeNotFound(w, reqPath)
		}
	}
}
