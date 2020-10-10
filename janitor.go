package main

import (
	"fmt"
	"github.com/jaywhyzed/slackJanitor/client"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/create_channel", createChannelHandler)

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

var _CALIFORNIA_LOCATION *time.Location

// CaliforniaLocation returns a time.Location representing California.
func CaliforniaLocation() *time.Location {
	if _CALIFORNIA_LOCATION != nil {
		return _CALIFORNIA_LOCATION
	}
	var err error
	_CALIFORNIA_LOCATION, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatalf("Got error loading location: %+v\n", err)
	}
	return _CALIFORNIA_LOCATION
}

// timeAsChannelName converts a time.Time to the name of a channel.
func timeAsChannelName(t time.Time) string {
	return t.Format("20060102")
}

// The name of the new channel, to be created.
func newChannel() string {
	return timeAsChannelName(time.Now().In(CaliforniaLocation()))
}

// The name of the old channel, to be archived.
func oldChannel() string {
	return timeAsChannelName(time.Now().AddDate(0, 0, -7).In(CaliforniaLocation()))
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
	json_str := client.ExecuteOrDie(client.CreateChannelRequest{Name: name}, &channel_resp)
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
	users_resp := client.UsersListResponse{}

	users := make([]client.User, 0)

	for ok := true; ok == true; ok = len(users_req.Cursor) > 0 {
		json_str := client.ExecuteOrDie(users_req, &users_resp)
		if users_resp.Ok == false {
			log.Fatalf("Error getting non-bot users:\n%s", json_str)
		}

		// log.Printf("Got raw response:\n%v\n", *json_str)

		for _, user := range users_resp.Members {
			log.Printf("Got member: %+v\n", user)
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
		client.ExecuteOrDie(channels_req, &channels_resp)
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
	is_cron := r.Header.Get("X-Appengine-Cron") //  , false)
	log.Printf("Called from Appengine-Cron: %v\n", is_cron)

	if is_cron != "true" {
		fmt.Fprintf(w, "This handler only accepts requests from Appengine Cron.\n")
		log.Printf("Called from non-cron: %+v", *r)
		log.Printf("Headers: %+v", r.Header)
		return
	}

	if r.URL.Path != "/create_channel" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprint(w, "Hello, World!\n")

	channel_resp := createChannelOrDie(newChannel())
	fmt.Fprintf(w, "Got ChannelCreate Response:%+v\n", channel_resp)
	log.Printf("Got ChannelCreate Response:%+v\n", channel_resp)

	channel := &channel_resp.Channel

	if channel_resp.Ok == false {
		log.Printf("Failed to create channel: %+v", channel_resp)
		fmt.Fprintf(w, "Failed to create channel: %+v", channel_resp)
		if channel_resp.Error == "name_taken" {
			fmt.Fprintf(w, "Channel already exists, fetching it...\n")
			log.Printf("Fetching existing channel...")
			channel = getChannelOrDie(newChannel())
			if channel != nil {
				fmt.Fprintf(w, "Fetched channel:\n%+v\n", *channel)
			} else {
				log.Fatal("Couldn't find channel!")
			}
		}
	}

	if channel == nil {
		log.Printf("Can't find the channel #%s!", newChannel())
		http.NotFound(w, r)
		return
	}

	log.Printf("Setting topic...")
	fmt.Fprintf(w, "Setting topic...\n")
	set_topic_resp := client.GenericResponse{}
	client.ExecuteOrDie(client.ChannelSetTopic{
		ChannelId: channel.Id,
		Topic:     "Zoom: " + os.Getenv("ZOOM_URL"),
	},
		set_topic_resp)

	fmt.Fprintf(w, "set topic response:\n%+v\n", set_topic_resp)
	log.Printf("set topic response:\n%+v", set_topic_resp)

	log.Printf("Getting Users")
	users := getNonBotUsersOrDie()

	log.Printf("Got Users %+v", users)
	fmt.Fprintf(w, "Got %d Users\n", len(users))

	invitation := client.ConversationInvite{Channel: channel.Id}

	for _, user := range users {
		invitation.Users = append(invitation.Users, user.Id)
	}

	invite_response := client.ChannelResponse{}
	log.Printf("Sending invitation...")
	fmt.Fprintf(w, "Sending invitation to new channel...\n")
	json_resp := client.ExecuteOrDie(invitation, &invite_response)
	if invite_response.Ok == false {
		// Invitation will fail if users are already added, not idempotent. Just ignore.
		log.Printf("Invitation failed! Ignoring.\n%+v\n%s", invite_response, json_resp)
		fmt.Fprintf(w, "Invitation failed! Ignoring.\n\n%s", json_resp)
	}

	post_resp := client.GenericResponse{}
	client.ExecuteOrDie(
		client.PostMessageRequest{
			ChannelId: channel.Id,
			Text: "Hello, welcome to today's channel.\n" +
				"As always, the Zoom link is " + os.Getenv("ZOOM_URL")},
		&post_resp)

	old_channel := getChannelOrDie(oldChannel())
	if old_channel == nil {
		log.Printf("Couldn't find old channel #%s, ignoring", oldChannel())
		fmt.Fprintf(w, "Couldn't find old channel #%s\n", oldChannel())
	} else {
		fmt.Fprintf(w, "Attempting to archive old channel.\n")
		log.Printf("Attempting to archive old channel.")
		archive_resp := client.GenericResponse{}
		client.ExecuteOrDie(client.ChannelArchiveRequest{ChannelId: old_channel.Id}, &archive_resp)

		if archive_resp.Ok == false {
			fmt.Fprintf(w, "Archive failed, ignoring:\n%+v", archive_resp)
			log.Printf("Archive failed, ignoring:\n%+v", archive_resp)
		} else {
			fmt.Fprintf(w, "Archive done.\n")
			log.Printf("Archive done.")
		}
	}
}
