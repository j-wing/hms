package hms

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
)

type routeHandler func(http.ResponseWriter, *http.Request, []string) *appError

var (
	indexTmpl = template.Must(getTemplate("index.html"))
)

type IndexTemplateParams struct {
	Path       string
	TargetURL  string
	Message    string
	CreatedURL string
	Host       string
	PastLinks  []Link
}

var shortenerRoutes = map[*regexp.Regexp]routeHandler{
	regexp.MustCompile("/([A-Z0-9-]+)[/]?$"): handleAutoShortURL,
	regexp.MustCompile("/([a-z]+[0-9]*)$"):   handleManualShortURL,
	regexp.MustCompile("/$"):                 handleChatIndex,
}

// Base handler for all requests handled by the URL shortenening/archiving code.
func ShortenerHandler(w http.ResponseWriter, r *http.Request) *appError {
	reqPath := r.URL.Path

	for routeRegex, handler := range shortenerRoutes {
		urlComponents := routeRegex.FindStringSubmatch(reqPath)
		if urlComponents != nil {
			return handler(w, r, urlComponents[1:])
		}
	}

	return &appError{nil, "Invalid URL", 404}
}

func handleChatIndex(w http.ResponseWriter, r *http.Request, params []string) *appError {
	if _, ok := handleUserAuth(w, r); !ok {
		return &appError{nil, "Unauthorized.", 403}
	}

	c := appengine.NewContext(r)

	var resultURL string

	if r.Method == "POST" {
		resultPath, err := createShortenedURL(r, -1)
		if err != nil {
			return &appError{err, err.Error(), http.StatusInternalServerError}
		}

		resultURL = fmt.Sprintf("http://%s/%s", r.Host, resultPath)
	}

	pastLinks := make([]Link, 0, 100)
	_, err := datastore.NewQuery("Link").Order("-Created").Limit(100).GetAll(c, &pastLinks)
	if err != nil {
		return &appError{err, err.Error(), http.StatusInternalServerError}
	}

	var message string
	path := r.FormValue("path")
	chatID := r.FormValue("chatID")

	if path != "" && r.Method == "GET" {
		_, err = getMatchingLinkChatString(c, chatID, path)
		if err != nil {
			message = "/" + path + " does not exist. Create it?"
		}
	}

	indexTmpl.Execute(w, IndexTemplateParams{
		Path:       path,
		TargetURL:  r.FormValue("target"),
		Host:       r.Host,
		PastLinks:  pastLinks,
		CreatedURL: resultURL,
		Message:    message,
	})
	return nil
}

func handleAutoShortURL(w http.ResponseWriter, r *http.Request, params []string) *appError {
	if _, ok := handleUserAuth(w, r); !ok {
		return &appError{nil, "Unauthorized.", 403}
	}

	urlPath := strings.TrimSpace(params[0])
	decodedKey := ShortURLDecode(urlPath)
	if decodedKey < 0 {
		return &appError{nil, "Invalid short url", 404}
	}

	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "Link", "", decodedKey, nil)

	log.Infof(c, "%d", key.IntID())

	var link Link
	err := datastore.Get(c, key, &link)
	if err == datastore.ErrNoSuchEntity {
		return &appError{err, "Invalid short url.", 404}
	} else if err != nil {
		return &appError{err, err.Error(), 500}
	}

	http.Redirect(w, r, link.TargetURL, http.StatusFound)
	return nil
}

func handleManualShortURL(w http.ResponseWriter, r *http.Request, params []string) *appError {
	if _, ok := handleUserAuth(w, r); !ok {
		return &appError{nil, "Unauthorized.", 403}
	}

	urlPath := params[0]
	strChatID := r.FormValue("chat")

	c := appengine.NewContext(r)
	target, err := getMatchingLinkChatString(c, strChatID, urlPath)
	if err != nil {
		if _, ok := err.(*strconv.NumError); ok {
			return &appError{nil, "Invalid FB chat ID", 401}
		} else {
			http.Redirect(w, r, fmt.Sprintf("/?path=%s&chatID=%s", urlPath, strChatID), http.StatusFound)
			return nil
		}
	}

	http.Redirect(w, r, target.TargetURL, http.StatusFound)
	return nil
}

func createShortenedURL(r *http.Request, chatID int64) (string, error) {
	path := r.FormValue("path")
	target := r.FormValue("target")

	if target == "" {
		return "", errors.New("empty target")
	} else {
		if !isValidPath(path) {
			return "", errors.New("invalid path")
		}

		parsedUrl, err := url.Parse(target)
		if err != nil {
			return "", err
		} else if parsedUrl.Scheme == "" {
			parsedUrl, err = url.Parse("http://" + target)
			if err != nil {
				return "", err
			}
		}

		if parsedUrl.Host == r.Host {
			return "", errors.New("Don't try to make redirect loops.")
		} else if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
			return "", errors.New("http[s] links only.")
		}

		c := appengine.NewContext(r)
		_, err = getMatchingLink(c, chatID, path)

		if err == nil {
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

		var chatKey *datastore.Key
		if chatID >= 0 {
			_, err = getOrCreateChat(c, chatID, &chatKey)
			if err != nil {
				return "", err
			}
		} else {
			chatKey = nil
		}

		u := Link{
			Path:      path,
			TargetURL: parsedUrl.String(),
			Creator:   creator,
			Created:   time.Now(),
			ChatKey:   chatKey,
		}

		finalPath := path

		err = datastore.RunInTransaction(c, func(tc context.Context) error {
			key := datastore.NewIncompleteKey(c, "Link", nil)
			newKey, err1 := datastore.Put(c, key, &u)
			if err1 != nil {
				return err1
			}

			if path == "" {
				newPath := ShortURLEncode(newKey.IntID())
				// Since this can be re-run multiple times,
				// this function has to be idempotent
				linkCopy := Link{
					Path:      newPath,
					TargetURL: u.TargetURL,
					Creator:   u.Creator,
					Created:   u.Created,
					ChatKey:   u.ChatKey,
				}
				_, err2 := datastore.Put(c, newKey, &linkCopy)
				if err2 != nil {
					return err2
				}
				finalPath = newPath
			}
			return nil
		}, nil)

		if err != nil {
			return "", err
		}

		return finalPath, nil
	}
}
