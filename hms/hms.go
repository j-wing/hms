package hms

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	//"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
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
	http.HandleFunc("/update_music_links", UpdateAllLinksWithMusicInfoHandler)
	http.HandleFunc("/add_chat", ChatAddHandler)
	http.HandleFunc("/backup", BackupLinksHandler)
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

func BackupLinksHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, r.URL.RequestURI())
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return
	} else {
		if !u.Admin {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("You're not an admin. Go away."))
		} else {
			w.Header().Set("Content-Type", "text/plain")
			results := datastore.NewQuery("Link").Order("-Created").Run(c)
			DELIM := "|||"
			var link Link
			for {
				_, err := results.Next(&link)
				if err == datastore.Done {
					break
				} else if err != nil {
					w.Write([]byte(err.Error()))
				} else {
					var chat Chat
					s := link.Path + DELIM + link.TargetURL + DELIM + link.Creator + DELIM
					s += strconv.FormatInt(link.Created.Unix(), 10) + DELIM
					if link.ChatKey != nil {
						err = datastore.Get(c, link.ChatKey, &chat)
						if err != nil {
							continue
						}
						s += strconv.FormatInt(chat.FacebookChatID, 10) + DELIM + chat.ChatName
					}
					w.Write([]byte(s + "\n"))
				}
			}
		}
	}
}
func ChatAddHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, r.URL.RequestURI())
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return
	} else {
		if !u.Admin {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("You're not an admin. Go away."))
		} else {
			name := r.FormValue("name")
			strChatID := r.FormValue("fbID")

			if name == "" || strChatID == "" {
				w.Write([]byte("You forgot a parameter."))
			}

			fbChatID, err := strconv.ParseInt(strChatID, 10, 64)
			if err != nil {
				w.Write([]byte("Chat ID has to be a number."))
			} else {
				chat := Chat{
					ChatName:       name,
					FacebookChatID: fbChatID,
				}
				dkey := datastore.NewIncompleteKey(c, "Chat", nil)
				_, err := datastore.Put(c, dkey, &chat)
				if err != nil {
					w.Write([]byte(fmt.Sprintf("error! %s", err.Error())))
				} else {
					w.Write([]byte("Success!"))

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
			w.WriteHeader(http.StatusForbidden)
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
				dkey := datastore.NewIncompleteKey(c, "APIKey", nil)
				_, err := datastore.Put(c, dkey, &apiKey)
				if err != nil {
					w.Write([]byte(fmt.Sprintf("error! %s", err.Error())))
				} else {
					w.Write([]byte(key))

				}
			}
		}
	}
}

func UpdateAllLinksWithMusicInfoHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		loginUrl, _ := user.LoginURL(c, r.URL.RequestURI())
		http.Redirect(w, r, loginUrl, http.StatusFound)
		return
	} else {
		if !u.Admin {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("You're not an admin. Go away."))
		} else {
			limit := 50
			q := datastore.NewQuery("Link").Limit(limit)

			converted := r.FormValue("converted")
			strProgressed := r.FormValue("progressed")
			cursorStr := r.FormValue("cursor")

			if cursorStr != "" {
				cursor, _ := datastore.DecodeCursor(cursorStr)
				q = q.Start(cursor)
			}

			t := q.Run(c)
			updated := 0
			runCount := 0
			for {
				var l Link
				key, err := t.Next(&l)
				if err != nil {
					if err == datastore.Done {
						break
					} else {
						log.Errorf(c, "Error fetching next link: %v", err)
						break
					}
				}

				if l.IsLikelyMusicLink() && l.MusicInfo.Title == "" {
					var info MusicInfo
					client := urlfetch.Client(c)
					params := url.Values{}
					params.Set("link", l.TargetURL)

					// TODO implement a task queue operation to fill in the info if this request fails.
					resp, err := client.Get("http://music.hms.space/get_music_info?" + params.Encode())
					if err != nil {
						log.Errorf(c, "Request for music info for %v failed. Error: %v", l.TargetURL, err.Error())
					} else {
						defer resp.Body.Close()
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							log.Errorf(c, "Failed to read body: %v", err.Error())
						} else {
							err = json.Unmarshal(body, &info)
							if err != nil {
								log.Errorf(c, "Failed to parse music response json: %v; json was %v", err.Error(), body)
							} else {
								l.MusicInfo = info
								datastore.Put(c, key, &l)
								updated += 1
							}
						}
					}
				}
				runCount += 1
			}

			total_updated := updated
			progressed := runCount

			if converted != "" {
				iConverted, _ := strconv.Atoi(converted)
				total_updated += iConverted
			}
			if strProgressed != "" {
				iProgressed, _ := strconv.Atoi(strProgressed)
				progressed += iProgressed

			}
			if cursor, err := t.Cursor(); err == nil && runCount != 0 {

				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(fmt.Sprintf("<a href='http://hms.space/update_music_links?progressed=%v&converted=%v&cursor=%v'>%v progressed, %v more converted (%v total)!</a>", progressed, total_updated, cursor, progressed, updated, total_updated)))
			} else {
				w.Write([]byte(fmt.Sprintf("Finished! %v total converted, %v total analyzed.", total_updated, progressed)))
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
*/
