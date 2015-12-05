package hms

import (
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

var (
	indexTmpl = template.Must(template.ParseFiles("index.html"))
)

const LETTERS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type IndexTemplateParams struct {
	Path       string
	TargetURL  string
	Message    string
	CreatedURL string
	Host       string
	PastLinks  []Link
}

type Link struct {
	Path      string
	TargetURL string
	Creator   string
	Created   time.Time
}

func (l *Link) FormatCreated() string {
	return l.Created.Add(time.Hour * -8).Format("3:04pm, Monday, January 2")
}

func makeLinkKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Link", "default_urlmatch", 0, nil)
}

func createRandomPath(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = LETTERS[rand.Intn(len(LETTERS))]
	}
	return string(b)

}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	http.HandleFunc("/add_api_key", APIKeyAddHandler)
	http.Handle("/api/", appHandler(APIHandler))
	http.HandleFunc("/add", QuickAddHandler)
	http.HandleFunc("/", ShortenerHandler)
}

func APIKeyAddHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, r.URL.RequestURI())
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return
	} else {
		if !u.Admin {
			w.Write([]byte("You're not an admin. Go away."))
		} else {
			key := createRandomPath(26)
			owner := r.FormValue("owner")

			if owner == "" {
				w.Write([]byte("You forgot a parameter."))
			} else {
				apiKey := APIKey{
					APIKey:     key,
					OwnerEmail: owner,
				}
				dkey := datastore.NewIncompleteKey(c, "APIKey", makeAPIKey(c))
				_, err := datastore.Put(c, dkey, &apiKey)
				if err != nil {
					w.Write([]byte(fmt.Sprintf("error! %s", err.Error())))
				} else {
					w.Write([]byte(fmt.Sprintf("success! Added key: %s", key)))

				}
			}
		}
	}
}

func QuickAddHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	if handleUserAuth(w, r) == nil {
		return
	}

	pastLinks, _ := getPastLinks(c, 100)

	resURL, err := createShortenedURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		fullURL := fmt.Sprintf("%v/%v", r.Host, resURL)
		indexTmpl.Execute(w, IndexTemplateParams{
			CreatedURL: fullURL,
			Host:       r.Host,
			PastLinks:  pastLinks,
		})
	}
}

func isValidPath(path string) bool {
	return !strings.Contains(path, "/")
}

func createShortenedURL(r *http.Request) (string, error) {
	path := r.FormValue("path")
	target := r.FormValue("target")

	if target == "" {
		return "", errors.New("empty target")
	} else {
		if path == "" {
			path = createRandomPath(4)
		} else if !isValidPath(path) {
			return "", errors.New("invalid path")
		}

		parsedUrl, err := url.Parse(target)
		if err != nil {
			return "", err
		} else if parsedUrl.Host == r.Host {
			return "", errors.New("Don't try to make redirect loops.")
		}

		if parsedUrl.Scheme == "" {
			parsedUrl.Scheme = "http"
		}

		c := appengine.NewContext(r)

		existingLinkCount, err := datastore.NewQuery("Link").Filter("Path =", path).Count(c)
		if existingLinkCount != 0 {
			return "", errors.New("There already exists a link with that path. ")
		}

		currUser := user.Current(c)
		var creator string
		if currUser == nil {
			creator = r.FormValue("creator")
			if creator == "" {
				return "", errors.New("No creator provided.")
			}
		} else {
			creator = currUser.Email
		}

		u := Link{
			Path:      path,
			TargetURL: parsedUrl.String(),
			Creator:   creator,
			Created:   time.Now(),
		}

		key := datastore.NewIncompleteKey(c, "Link", makeLinkKey(c))
		_, err = datastore.Put(c, key, &u)
		if err != nil {
			return "", err
		}
		return path, nil
	}
}

func writeNotFound(w http.ResponseWriter, path string) {
	indexTmpl.Execute(w, IndexTemplateParams{
		Path:    path[1:],
		Message: "No entry for that path. Add one?",
	})
}

func getMatchingLink(requestPath string, c appengine.Context) ([]Link, error) {
	match := make([]Link, 0, 1)
	_, err := datastore.NewQuery("Link").Filter("Path =", requestPath[1:]).Limit(1).GetAll(c, &match)
	if err != nil {
		return match, err
	}
	return match, nil
}

func getPastLinks(c appengine.Context, limit int) ([]Link, error) {
	pastLinks := make([]Link, 0, 100)
	_, err := datastore.NewQuery("Link").Order("-Created").Limit(100).GetAll(c, &pastLinks)
	return pastLinks, err
}

func isAuthorizedUser(user user.User) bool {
	_, ok := ALLOWED_EMAILS[user.Email]
	return ok
}

func handleUserAuth(w http.ResponseWriter, r *http.Request) *user.User {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, "/")
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return nil
	} else if !isAuthorizedUser(*u) {
		w.Write([]byte("Email not in authorized list. Message me to get access. "))
		return nil
	}

	return u
}

func ShortenerHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	c := appengine.NewContext(r)

	pastLinks, err := getPastLinks(c, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if reqPath == "/" {
		if handleUserAuth(w, r) == nil {
			return
		}
		if r.Method == "GET" {
			indexTmpl.Execute(w, IndexTemplateParams{
				Path:      r.FormValue("path"),
				TargetURL: r.FormValue("target"),
				Host:      r.Host,
				PastLinks: pastLinks,
			})
		} else if r.Method == "POST" {
			resURL, err := createShortenedURL(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				fullURL := fmt.Sprintf("%v/%v", r.Host, resURL)
				indexTmpl.Execute(w, IndexTemplateParams{
					CreatedURL: fullURL,
					Host:       r.Host,
					PastLinks:  pastLinks,
				})
			}
		}
	} else {
		link_arr, err := getMatchingLink(reqPath, c)

		if err == nil {
			if len(link_arr) > 0 {
				http.Redirect(w, r, link_arr[0].TargetURL, http.StatusFound)
			} else {
				writeNotFound(w, reqPath)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
