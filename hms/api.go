package hms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

const API_BATCH_AMT = 100

type AddSuccessResponse struct {
	Success   bool
	ResultURL string
}

type ResolveResponse struct {
	Success bool
	Result  *Link
	Chat    *Chat
	Error   string
}

type ListResponse struct {
	Success bool
	Links   []Link
	Chat    *Chat
	Error   string
}

type RemoveResponse struct {
	Success    bool
	NumRemoved int
	Error      string
}

type apiHandler func(http.ResponseWriter, *http.Request, APIKey) *appError

var apiRoutes = map[string]apiHandler{
	"/api/add":     handleAdd,
	"/api/resolve": handleResolve,
	"/api/list":    handleList,
	"/api/remove":  handleRemove,
}

func handleAdd(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "POST" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	fbChatID := int64(-1)

	strChatID := r.FormValue("chatID")

	if strChatID != "" {
		var err error
		fbChatID, err = strconv.ParseInt(strChatID, 10, 64)
		if err != nil {
			return &appError{err, "Invalid chat ID: " + err.Error(), 400}
		}
	} else {
		fbChatID = -1
	}

	resURL, err := createShortenedURL(r, fbChatID)
	if err != nil {
		// TODO handle this case better by distinguishing between
		// bad requests and e.g. datastore errors
		return &appError{err, err.Error(), 400}
	}

	absResURL := fmt.Sprintf("http://%s/%s", r.Host, resURL)
	if strChatID != "" {
		absResURL += "?chatID=" + strChatID
	}

	resp := &AddSuccessResponse{true, absResURL}
	respJSON, _ := json.Marshal(resp)
	w.Write(respJSON)
	return nil
}

func handleResolve(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "GET" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	reqPath := r.FormValue("path")
	if reqPath == "" {
		return &appError{nil, "The `path` parameter is required. ", 401}
	}
	c := appengine.NewContext(r)
	linkResult, err := getMatchingLinkChatString(c, r.FormValue("chatID"), reqPath)

	var resp *ResolveResponse

	if err != nil {
		resp = &ResolveResponse{false, nil, nil, "Not Found"}
	} else {
		var resultChat Chat

		if linkResult.ChatKey != nil {
			err = datastore.Get(c, linkResult.ChatKey, &resultChat)
		}
		resp = &ResolveResponse{true, linkResult, &resultChat, ""}
	}
	respJSON, _ := json.Marshal(resp)
	w.Write(respJSON)
	return nil
}

func handleList(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "GET" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	sLimit := r.FormValue("limit")
	sOffset := r.FormValue("offset")
	strChatID := r.FormValue("chatID")

	fbChatID := int64(-1)

	var err error
	if strChatID != "" {
		fbChatID, err = strconv.ParseInt(strChatID, 10, 64)
		if err != nil {
			return &appError{nil, "Bad chat ID", 400}
		}
	}

	var limit, offset int

	var n int64
	var err1, err2 error

	if sLimit != "" {
		n, err1 = strconv.ParseInt(sLimit, 10, 32)
		limit = int(n)
	} else {
		limit = API_BATCH_AMT
	}

	if sOffset != "" {
		n, err2 = strconv.ParseInt(sOffset, 10, 32)
		offset = int(n)
	} else {
		offset = 0
	}

	if err1 != nil || err2 != nil {
		if err1 != nil {
			err = err1
		} else {
			err = err2
		}
		return &appError{err, "Bad limit or offset: " + err.Error(), 400}
	}

	if limit > API_BATCH_AMT {
		limit = API_BATCH_AMT
	}
	if offset > limit {
		offset = 0
	}

	c := appengine.NewContext(r)
	var chat *Chat
	var chatKey *datastore.Key
	if fbChatID != -1 {
		chatResults := make([]Chat, 0, 1)
		chatKeys, err := datastore.NewQuery("Chat").
			Filter("FacebookChatID =", fbChatID).Limit(1).GetAll(c, &chatResults)
		if err != nil {
			return &appError{err, "Datastore error: " + err.Error(), 500}
		} else if len(chatKeys) == 0 {
			return &appError{nil, "No matching chat ID", 404}
		}

		chatKey = chatKeys[0]
		chat = &chatResults[0]
	} else {
		chat = nil
		chatKey = nil
	}

	results := make([]Link, 0)
	q := datastore.NewQuery("Link").
		Filter("ChatKey =", chatKey).
		Order("-Created").Offset(offset).Limit(limit)
	_, err = q.GetAll(c, &results)

	if err != nil {
		return &appError{err, "Datastore error: " + err.Error(), 500}
	}

	resp := ListResponse{
		true,
		results,
		chat,
		"",
	}

	respJSON, _ := json.Marshal(resp)
	w.Write(respJSON)
	return nil
}

func handleRemove(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "DELETE" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	c := appengine.NewContext(r)

	strChatID := r.FormValue("chatID")
	rmPath := r.FormValue("path")

	if rmPath == "" {
		return &appError{nil, "Missing path.", 401}
	}

	var fbChatID int64 = -1
	var chatKey *datastore.Key
	var err error

	if strChatID != "" {
		fbChatID, err = strconv.ParseInt(strChatID, 10, 64)
		if err != nil {
			return &appError{nil, "Bad chat ID", 400}
		}
		chatKeys, err := datastore.NewQuery("Chat").Filter("FacebookChatID =", fbChatID).
			KeysOnly().GetAll(c, nil)

		if err != nil {
			return &appError{err, "Datastore error: " + err.Error(), 400}
		} else if len(chatKeys) == 0 {
			return &appError{nil, "Bad chat ID", 400}
		}

		chatKey = chatKeys[0]
	} else {
		chatKey = nil
	}

	deleted := make([]Link, 0)
	keysToRemove, err := datastore.NewQuery("Link").
		Filter("Path =", rmPath).Filter("ChatKey =", chatKey).GetAll(c, &deleted)

	if len(keysToRemove) != 0 {
		newKeys := make([]*datastore.Key, len(keysToRemove))
		for i := range keysToRemove {
			newKeys[i] = datastore.NewIncompleteKey(c, "DeletedLink", nil)
		}

		err = datastore.RunInTransaction(c, func(tc context.Context) (err error) {
			_, err = datastore.PutMulti(c, newKeys, deleted)
			if err != nil {
				return
			}
			err = datastore.DeleteMulti(c, keysToRemove)
			return
		}, nil)

	}

	var resp RemoveResponse
	if err != nil {
		resp = RemoveResponse{
			false,
			0,
			err.Error(),
		}
	} else {
		resp = RemoveResponse{
			true,
			len(keysToRemove),
			"",
		}
	}

	respJSON, _ := json.Marshal(resp)
	w.Write(respJSON)
	return nil
}

// General handler function for all requests to the API
// In addition to calling the handler function according to the API routes,
// verifies that a valid API key was provided as a parameter, and
// sets the response content-type to JSON
func APIHandler(w http.ResponseWriter, r *http.Request) *appError {
	apiKey := r.FormValue("apiKey")
	if apiKey == "" {
		return &appError{nil, "Invalid API Key", 401}
	}

	c := appengine.NewContext(r)
	results := make([]APIKey, 0, 1)
	_, err := datastore.NewQuery("APIKey").Filter("APIKey =", apiKey).GetAll(c, &results)
	if err != nil {
		return &appError{err, "Error validating API key", 500}
	} else if len(results) == 0 {
		return &appError{nil, "Invalid API key.", 401}
	}

	apiKeyStruct := results[0]
	handler, ok := apiRoutes[r.URL.Path]
	if !ok {
		return &appError{nil, fmt.Sprintf("No API handler for %s", r.URL.Path), 404}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return handler(w, r, apiKeyStruct)
}
