package hms

import (
	"errors"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

// Used to represent a single Facebook chat
type Chat struct {
	ChatName       string
	FacebookChatID int64
}

func getOrCreateChat(c context.Context, fbChatID int64, keyBuf **datastore.Key) (*Chat, error) {
	results := make([]Chat, 0, 1)
	keys, err := datastore.NewQuery("Chat").
		Filter("FacebookChatID =", fbChatID).Limit(1).GetAll(c, &results)
	if err != nil {
		return nil, err
	}

	var resultChat *Chat
	var resultKey *datastore.Key

	if len(keys) == 0 {
		incKey := datastore.NewIncompleteKey(c, "Chat", nil)
		resultChat = &Chat{
			FacebookChatID: fbChatID,
			ChatName:       "",
		}
		resultKey, err = datastore.Put(c, incKey, resultChat)
		if err != nil {
			return nil, err
		}
	} else {
		resultKey = keys[0]
		resultChat = &results[0]
	}

	if keyBuf != nil {
		*keyBuf = resultKey
	}

	return resultChat, nil
}

// Represents a single shared link
type Link struct {
	Path      string
	TargetURL string
	Creator   string
	Created   time.Time
	ChatKey   *datastore.Key `json:"-"`
	MusicInfo *MusicInfo
}

type MusicInfo struct {
	Artist     string      `json:"artist"`
	Genres     []string    `json:"genres"`
	SubGenres  []string    `json:"subgenres"`
	SourceType MusicSource `json:"sourceType,string"`
	Title      string      `json: "title"`
}

// Used by templates to format the Link struct's created field.
func (l *Link) FormatCreated() string {
	return l.Created.Add(time.Hour * -8).Format("3:04pm, Monday, January 2")
}

func getMatchingLink(c context.Context, fbChatID int64, path string) (*Link, error) {
	var chatKey *datastore.Key
	chatKey = nil

	if fbChatID >= 0 {
		chatKeys, err := datastore.NewQuery("Chat").Filter("FacebookChatID =", fbChatID).KeysOnly().GetAll(c, nil)
		if err != nil {
			return nil, err
		} else if len(chatKeys) == 0 {
			return nil, errors.New("No matching chat key")
		}

		chatKey = chatKeys[0]
	}

	match := make([]Link, 0, 1)
	_, err := datastore.NewQuery("Link").Filter("Path =", path).Filter("ChatKey =", chatKey).Limit(1).GetAll(c, &match)
	if err != nil {
		return nil, err
	} else if len(match) == 0 {
		return nil, errors.New("No matching link")
	}
	return &match[0], nil
}

func getMatchingLinkChatString(c context.Context, strFbChatID string, path string) (*Link, error) {
	var fbChatID int64 = -1
	var err error
	if strFbChatID != "" {
		fbChatID, err = strconv.ParseInt(strFbChatID, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return getMatchingLink(c, fbChatID, path)
}

type APIKey struct {
	APIKey     string
	OwnerEmail string
	Created    time.Time
	valid      bool
}
