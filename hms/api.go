package hms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

const API_MAX_LIMIT = 50

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

type apiHandler func(http.ResponseWriter, *http.Request, APIKey) *appError

var apiRoutes = map[string]apiHandler{
	"/api/add":     handleAdd,
	"/api/resolve": handleResolve,
	"/api/list":    handleList,
}

func handleAdd(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "POST" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	fbChatID := int64(-1)

	strChatID := r.FormValue("chatID")

	fbChatID, err := strconv.ParseInt(strChatID, 10, 64)
	if strChatID != "" && err != nil {
		return &appError{err, "Invalid chat ID: " + err.Error(), 400}
	}

	resURL, err := createShortenedURL(r, fbChatID)
	if err != nil {
		// TODO handle this case better by distinguishing between
		// bad requests and e.g. datastore errors
		return &appError{err, err.Error(), 400}
	}

	absResURL := fmt.Sprintf("http://%s/%s", r.Host, resURL)
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

	var limit int
	var offset int

	var n int64
	var err1 error
	var err2 error

	if sLimit != "" {
		n, err1 = strconv.ParseInt(sLimit, 10, 32)
		limit = int(n)
	} else {
		limit = API_MAX_LIMIT
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

	if limit > API_MAX_LIMIT {
		limit = API_MAX_LIMIT
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

	results := make([]Link, 0, limit)
	_, err = datastore.NewQuery("Link").
		Filter("ChatKey =", chatKey).
		Order("-Created").
		Limit(limit).Offset(offset).GetAll(c, &results)

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
	return handler(w, r, apiKeyStruct)
}
