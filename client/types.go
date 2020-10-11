package client

import (
	"log"
	"net/url"
)

type Request interface {
	URL() string
	Verb() string
}

type CreateChannelRequest struct {
	Name string `json:"name"`
}

type ConversationInvite struct {
	Channel string   `json:"channel"`
	Users   []string `json:"users"`
}

type Channel struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ChannelResponse struct {
	Ok      bool    `json:"ok"`
	Channel Channel `json:"channel"`

	Error       string `json:"error"`
	ErrorDetail string `json:"detail"`
	Warning string `json:"warning"`
}

type ChannelArchiveRequest struct {
	ChannelId string `json:"channel"`
}

type GenericResponse struct {
	Ok      bool   `json:"ok"`
	Warning string `json:"warning"`
	Error   string `json:"error"`
}

type ChannelSetTopic struct {
	ChannelId string `json:"channel"`
	Topic     string `json:"topic"`
}

type Block struct {
	Type   string `json:"type"`
	CallId string `json:"call_id"`
}


type PostMessageRequest struct {
	ChannelId string `json:"channel"`
	Text      string `json:"text"`
	Blocks    []Block `json:"blocks"`
}

func (r PostMessageRequest) URL() string {
	return "https://slack.com/api/chat.postMessage"
}

func (r PostMessageRequest) Verb() string {
	return "POST"
}

func (r ChannelSetTopic) URL() string {
	return "https://slack.com/api/conversations.setTopic"
}

func (r ChannelSetTopic) Verb() string {
	return "POST"
}

func (r ChannelArchiveRequest) URL() string {
	return "https://slack.com/api/conversations.archive"
}

func (r ChannelArchiveRequest) Verb() string {
	return "POST"
}

func (r CreateChannelRequest) URL() string {
	return "https://slack.com/api/conversations.create"
}
func (r CreateChannelRequest) Verb() string {
	return "POST"
}

func (r ConversationInvite) URL() string {
	return "https://slack.com/api/conversations.invite"
}
func (r ConversationInvite) Verb() string {
	return "POST"
}

type ChannelListRequest struct {
	Cursor string
}

type ChannelListResponse struct {
	Ok       bool             `json:"ok"`
	Channels []Channel        `json:"channels"`
	Metadata ResponseMetadata `json:"response_metadata"`
	Error    string           `json:"error"`
}

// We use Call for both the request creating the call, and also part of the
// CallResponse.
type Call struct {
	// Return-only
	Id string `json:"id"`

	// Required for requests
	ExternalUniqueId string `json:"external_unique_id"`
	JoinUrl          string `json:"join_url"`

	// Optional
	StartTimeUnix     int64    `json:"date_start"`
	DesktopAppJoinUrl string `json:"desktop_app_join_url"`
	ExternalDisplayId string `json:"external_display_id"`
	Title             string `json:"title"`
}

type CallResponse struct {
	Ok      bool   `json:"ok"`
	Call    Call   `json:"call"`
	Warning string `json:"warning"`
	Error   string `json:"error"`
}

func (r Call) Verb() string {
	return "POST"
}
func (r Call) URL() string {
	return "https://slack.com/api/calls.add"
}

func (r ChannelListRequest) Verb() string {
	return "GET"
}

func (r ChannelListRequest) URL() string {
	u, err := url.Parse("https://slack.com/api/conversations.list")
	if err != nil {
		log.Fatal(err)
	}
	query := u.Query()
	query.Set("exclude_archived", "true")
	query.Set("types", "public_channel")
	if len(r.Cursor) > 0 {
		query.Set("cursor", r.Cursor)
	}
	u.RawQuery = query.Encode()

	return u.String()
}

func (r UsersListRequest) Verb() string {
	return "GET"
}

type UsersListRequest struct {
	// These don't encode to JSON, since this isn't a POST request.
	Cursor string
	Limit  string
}

type User struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
	IsBot   bool   `json:"is_bot"`
}

type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

type UsersListResponse struct {
	Ok       bool             `json:"ok"`
	Members  []User           `json:"members"`
	Metadata ResponseMetadata `json:"response_metadata"`

	Error       string `json:"error"`
	ErrorDetail string `json:"detail"`
}

func (r UsersListRequest) URL() string {
	u, err := url.Parse("https://slack.com/api/users.list")
	if err != nil {
		log.Fatal(err)
	}
	query := u.Query()
	if len(r.Cursor) > 0 {
		query.Set("cursor", r.Cursor)
	}
	if len(r.Limit) > 0 {
		query.Set("limit", r.Limit)
	}
	u.RawQuery = query.Encode()

	return u.String()
}
