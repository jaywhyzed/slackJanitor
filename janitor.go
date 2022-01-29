package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/jaywhyzed/slackJanitor/client"
)

func main() {

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/create_channel", createChannelHandler)
	http.HandleFunc("/post_call", postCallHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// A time.Location representing California.
var CaliforniaLocation *time.Location
var slackClient client.Client

// Initialize the client if necessary, and all Execute.
func Execute(req client.Request, resp interface{}) (string, error) {
	// Clear the response.
	p := reflect.ValueOf(resp).Elem()
	p.Set(reflect.Zero(p.Type()))

	if slackClient == nil {
		token := os.Getenv("SLACK_BOT_USER_TOKEN")
		if len(token) == 0 {
			log.Fatal("Missing token!")
		}
		slackClient = client.NewClient(token)
	}
	return slackClient.Execute(req, resp)
}

// Like Execute() but dies on underlying failure.
func ExecuteOrDie(req client.Request, resp interface{}) string {
	respText, err := Execute(req, resp)
	if err != nil {
		log.Fatalf("Encountered error: %s\nHandling request:\n%+v\nResponse text:\n%s",
			err, req, respText)
	}
	return respText
}

func init() {
	var err error
	CaliforniaLocation, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatalf("Got error loading location: %+v\n", err)
	}
}

// timeAsChannelName converts a time.Time to the name of a channel.
func timeAsChannelName(t time.Time) string {
	return t.Format("20060102")
}

// The name of the new channel, to be created.
func newChannelName() string {
	return timeAsChannelName(time.Now().In(CaliforniaLocation))
}

// The name of the old channel, to be archived.
func oldChannelName() string {
	return timeAsChannelName(time.Now().AddDate(0, 0, -7).In(CaliforniaLocation))
}

// indexHandler responds to requests with our greeting.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Hello, World!\n")
}

// createChannelOrDie attempts to create a Slack channel with the given name,
// and returns the API response.
// Dies on HTTP error.
func createChannelOrDie(name string) client.ChannelResponse {
	var channel_resp client.ChannelResponse
	json_str := ExecuteOrDie(client.CreateChannelRequest{Name: name}, &channel_resp)
	if channel_resp.Ok == false {
		log.Printf("Error creating channel? Response:\n%s\n", json_str)
	}
	log.Printf("Created Channel:\n%v", channel_resp)
	return channel_resp
}

// getNonBotUsersOrDie calls the Slack API to get a list of non-bot Users.
// Dies on HTTP error.
func getNonBotUsersOrDie() []client.User {
	users_req := client.UsersListRequest{}
	users := make([]client.User, 0)

	for ok := true; ok == true; ok = len(users_req.Cursor) > 0 {
		var users_resp client.UsersListResponse
		json_str := ExecuteOrDie(users_req, &users_resp)
		if users_resp.Ok == false {
			log.Fatalf("Error getting non-bot users:\n%s", json_str)
		}

		for _, user := range users_resp.Members {
			if user.IsBot == false && user.Deleted == false {
				users = append(users, user)
			}
		}

		users_req.Cursor = users_resp.Metadata.NextCursor
	}

	return users
}

// May return nil if channel not found
func getChannelOrDie(name string) *client.Channel {
	channels_req := client.ChannelListRequest{}
	channels_resp := client.ChannelListResponse{}

	for ok := true; ok == true; ok = len(channels_req.Cursor) > 0 {
		log.Printf("Executing ChannelListRequest...")
		ExecuteOrDie(channels_req, &channels_resp)
		if channels_resp.Ok != true {
			log.Fatalf("Channels resp error:\n%+v", channels_resp)
		}
		for _, channel := range channels_resp.Channels {
			if channel.Name == name {
				return &channel
			}
		}
		channels_req.Cursor = channels_resp.Metadata.NextCursor
	}

	log.Printf("Couldn't find channel #%s", name)
	return nil
}

// createChannelHandler handles the /create_channel URL.
// Create a new channel.
// Set a topic.
// Add all non bot users to the new channel.
// Archive the old channel.
func createChannelHandler(w http.ResponseWriter, r *http.Request) {
	is_cron := r.Header.Get("X-Appengine-Cron")
	log.Printf("Called from Appengine-Cron: %v\n", is_cron)

	if is_cron != "true" {
		log.Printf("Called from non-cron: %+v", *r)
		log.Printf("Headers: %+v", r.Header)
		http.Error(w, "Only accepts calls from AppEngine Cron.\n", 400)
		return
	}

	if r.URL.Path != "/create_channel" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprint(w, "Hello, World!\n")

	channel_resp := createChannelOrDie(newChannelName())
	fmt.Fprintf(w, "Got ChannelCreate Response:%+v\n", channel_resp)

	channel := &channel_resp.Channel

	if channel_resp.Ok == false {
		log.Printf("Failed to create channel: %+v", channel_resp)
		fmt.Fprintf(w, "Failed to create channel: %+v", channel_resp)
		if channel_resp.Error == "name_taken" {
			fmt.Fprintf(w, "Channel already exists, fetching it...\n")
			log.Printf("Fetching existing channel...")
			channel = getChannelOrDie(newChannelName())
			if channel != nil {
				fmt.Fprintf(w, "Fetched channel:\n%+v\n", *channel)
			} else {
				log.Printf("Can't find the channel #%s!", newChannelName())
				http.NotFound(w, r)
				return
			}
		}
	}

	log.Printf("Setting topic...")
	fmt.Fprintf(w, "Setting topic...\n")
	set_topic_resp := client.GenericResponse{}
	ExecuteOrDie(client.ChannelSetTopicRequest{
		ChannelId: channel.Id,
		Topic:     "Video Call: " + os.Getenv("VC_URL"),
	},
		&set_topic_resp)
	if !set_topic_resp.Ok {
		log.Printf("Failed to set topic.")
	}

	log.Printf("Getting Users")
	users := getNonBotUsersOrDie()

	log.Printf("Got %d Users", len(users))

	invitation := client.ConversationInvite{ChannelId: channel.Id}

	for _, user := range users {
		invitation.Users = append(invitation.Users, user.Id)
	}

	invite_response := client.ChannelResponse{}
	fmt.Fprintf(w, "Sending invitation to new channel...\n")
	json_resp := ExecuteOrDie(invitation, &invite_response)
	if !invite_response.Ok {
		// Invitation will fail if users are already added, not idempotent. Just ignore.
		log.Printf("Invitation failed! Ignoring.\n%+v\n%s", invite_response, json_resp)
	}

	post_resp := client.GenericResponse{}
	ExecuteOrDie(
		client.PostMessageRequest{
			ChannelId: channel.Id,
			Text:      fmt.Sprintf("Hello, welcome to today's channel.\nOur new video call link is %s", os.Getenv("VC_URL"))},
		&post_resp)

	old_channel := getChannelOrDie(oldChannelName())
	if old_channel == nil {
		fmt.Fprintf(w, "Couldn't find old channel #%s\n", oldChannelName())
	} else {
		fmt.Fprintf(w, "Attempting to archive old channel.\n")
		archive_resp := client.GenericResponse{}
		ExecuteOrDie(client.ChannelArchiveRequest{ChannelId: old_channel.Id}, &archive_resp)

		if !archive_resp.Ok {
			log.Printf("Archive failed, ignoring:\n%+v", archive_resp)
		} else {
			log.Printf("Archive done.")
		}
	}
}

func todayAtSixThirty() time.Time {
	year, month, day := time.Now().In(CaliforniaLocation).Date()
	return time.Date(year, month, day, 18, 30, 0, 0, CaliforniaLocation)
}

// postCallHandler handles the /post_call URL.
// Create a Call object for the video call.
// Get the new Channel.
// Post the Call to the Channel.
func postCallHandler(w http.ResponseWriter, r *http.Request) {
	is_cron := r.Header.Get("X-Appengine-Cron")
	log.Printf("Called from Appengine-Cron: %v\n", is_cron)

	if is_cron != "true" {
		log.Printf("Called from non-cron: %+v", *r)
		log.Printf("Headers: %+v", r.Header)
		http.Error(w, "Only accepts calls from AppEngine Cron.\n", 400)
		return
	}

	if r.URL.Path != "/post_call" {
		http.NotFound(w, r)
		return
	}

	call := client.Call{
		ExternalUniqueId:  newChannelName(),
		JoinUrl:           os.Getenv("VC_URL"),
		ExternalDisplayId: os.Getenv("VC_CALL_ID"),
		Title:             "Game Time!",
		StartTimeUnix:     todayAtSixThirty().Unix(),
	}

	var callResp client.CallResponse
	ExecuteOrDie(call, &callResp)
	if !callResp.Ok {
		log.Fatalf("Error in call:\n%+v\n\nRequest was:\n%+v", callResp, call)
	}

	var postResp client.GenericResponse
	ExecuteOrDie(
		client.PostMessageRequest{
			ChannelId: getChannelOrDie(newChannelName()).Id,
			Text:      "Join the Video Call",
			Blocks: []client.Block{
				client.Block{Type: "call", CallId: callResp.Call.Id},
			},
		},
		&postResp)
	if !postResp.Ok {
		log.Fatalf("Error posting message:\n%+v", postResp)
	}
}
