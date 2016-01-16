package hms

import (
	//"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

type AddSuccessResponse struct {
	Success   bool
	ResultURL string
}

type ResolveResponse struct {
	Success bool
	Result  Link
}

type apiHandler func(http.ResponseWriter, *http.Request, APIKey) *appError

var apiRoutes = map[string]apiHandler{
	"/api/add":     handleAdd,
	"/api/resolve": handleResolve,
}

func handleAdd(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "POST" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}
	return nil /*
		resURL, err := createShortenedURL(r)
		if err != nil {
			// TODO handle this case better by distinguishing between
			// bad requests and e.g. datastore errors
			return &appError{err, err.Error(), 400}
		}

		absResURL := fmt.Sprintf("http://%s/%s", r.Host, resURL)
		resp := &AddSuccessResponse{true, absResURL}
		respJSON, _ := json.Marshal(resp)
		w.Write(respJSON)
		return nil*/
}

func handleResolve(w http.ResponseWriter, r *http.Request, apiKey APIKey) *appError {
	if r.Method != "GET" {
		return &appError{nil, fmt.Sprintf("Invalid request method: %s", r.Method), 401}
	}

	reqPath := r.FormValue("path")
	if reqPath == "" {
		return &appError{nil, "The `path` parameter is required. ", 401}
	}
	//c := appengine.NewContext(r)
	return nil /*
		linkResults, err := getMatchingLink(reqPath, c)

		var resp *ResolveResponse

		if err != nil || len(linkResults) != 1 {
			resp = &ResolveResponse{false, Link{}}
		} else {
			resp = &ResolveResponse{true, linkResults[0]}
		}
		respJSON, _ := json.Marshal(resp)
		w.Write(respJSON)
		return nil*/
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
