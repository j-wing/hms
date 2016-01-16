package hms

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	//"net/url"
	"strings"
	"time"

	//"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
)

type appError struct {
	Error   error
	Message string
	Code    int
}
type appHandler func(http.ResponseWriter, *http.Request) *appError

var templateBaseDir = getTemplateBaseDir()
var defaultErrTmpl = template.Must(getTemplate("err_default.html"))

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	http.HandleFunc("/add_api_key", APIKeyAddHandler)
	http.Handle("/api/", appHandler(APIHandler))
	http.Handle("/", appHandler(ShortenerHandler))
	//http.HandleFunc("/add", QuickAddHandler)
	//http.HandleFunc("/", ShortenerHandler)
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		c := appengine.NewContext(r)
		if e.Code == 500 {
			log.Errorf(c, "error recorded: %v; message: %v", e.Error, e.Message)
			http.Error(w, e.Message, e.Code)
		} else {
			if strings.HasPrefix(r.URL.Path, "/api") {
				asJson, _ := json.Marshal(e)
				http.Error(w, string(asJson), e.Code)
			} else {
				w.WriteHeader(e.Code)
				errTmpl, err := getErrorTemplate(e)
				if err != nil {
					defaultErrTmpl.Execute(w, e)
				} else {
					errTmpl.Execute(w, e)
				}
			}
		}
	}
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
			key := randomString(26)
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

/*
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

*/
