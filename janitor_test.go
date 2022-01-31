package janitor

import (
	"github.com/golang/mock/gomock"
	"github.com/jaywhyzed/slackJanitor/client"
	"github.com/jaywhyzed/slackJanitor/client/mocks"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestIndexHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IndexHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf(
			"unexpected status: got (%+v) want (%+v)",
			status,
			http.StatusOK,
		)
	}

	t.Logf("Returned body:\n%v", rr.Body.String())
}

func TestIndexHandlerNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/404", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(IndexHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusNotFound,
		)
	}
}

// Get the MockClient, and set it for the real handler to use.
func getClient(t *testing.T) *mocks.MockClient {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mocks.NewMockClient(mockCtrl)
	slackClient = mockClient
	return mockClient
}

func TestCreateChannelWithoutCronHeader(t *testing.T) {
	mockClient := getClient(t)
	mockClient.EXPECT().Execute(gomock.Any(), gomock.Any()).Return("", nil).Times(0)

	req, err := http.NewRequest("GET", "/create_channel", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateChannelHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusBadRequest,
		)
		t.Errorf("Returned body:\n%v", rr.Body.String())

	}
}

func TestPostCallWithoutCronHeader(t *testing.T) {
	mockClient := getClient(t)
	mockClient.EXPECT().Execute(gomock.Any(), gomock.Any()).Return("", nil).Times(0)

	req, err := http.NewRequest("GET", "/post_call", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PostCallHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusBadRequest,
		)
		t.Errorf("Returned body:\n%v", rr.Body.String())

	}
}

func TestCreateChannel(t *testing.T) {
	mockClient := getClient(t)

	// Set the mocks.

	gomock.InOrder(
		// First, Create the channel.
		mockClient.EXPECT().Execute(
			/*req=*/ client.CreateChannelRequest{Name: newChannelName()},
			/*resp=*/ gomock.AssignableToTypeOf(&client.ChannelResponse{})).DoAndReturn(
			func(req client.CreateChannelRequest,
				resp *client.ChannelResponse) (string, error) {
				resp.Ok = true
				resp.Channel.Id = "newchannelid"
				resp.Channel.Name = req.Name
				return "raw json haha", nil
			}).Times(1),
		// Set the Channel's Topic.
		mockClient.EXPECT().Execute(
			/*req=*/ client.ChannelSetTopicRequest{
				ChannelId: "newchannelid", Topic: "Video Call: http://zoom"},
			/*resp=*/ gomock.AssignableToTypeOf(&client.GenericResponse{})).DoAndReturn(
			func(req client.ChannelSetTopicRequest,
				resp *client.GenericResponse) (string, error) {
				resp.Ok = true
				return "raw json", nil
			}).Times(1),
		// Get all the users.
		mockClient.EXPECT().Execute(
			/*req=*/ client.UsersListRequest{},
			/*resp=*/ gomock.AssignableToTypeOf(&client.UsersListResponse{})).DoAndReturn(
			func(req client.UsersListRequest,
				resp *client.UsersListResponse) (string, error) {
				resp.Ok = true
				resp.Members = []client.User{
					client.User{Id: "123", Name: "User1", Deleted: false, IsBot: false},
					client.User{Id: "1337", Name: "Bot1", Deleted: false, IsBot: true},
					client.User{Id: "000", Name: "DeletedUser", Deleted: true, IsBot: false},
				}
				resp.Metadata.NextCursor = "contToken"
				return "raw json", nil
			}).Times(1),
		mockClient.EXPECT().Execute(
			/*req=*/ client.UsersListRequest{Cursor: "contToken"},
			/*resp=*/ gomock.AssignableToTypeOf(&client.UsersListResponse{})).DoAndReturn(
			func(req client.UsersListRequest,
				resp *client.UsersListResponse) (string, error) {
				resp.Ok = true
				resp.Members = []client.User{
					client.User{Id: "456", Name: "User2", Deleted: false, IsBot: false},
					client.User{Id: "789", Name: "User3", Deleted: false, IsBot: false},
				}
				return "raw json", nil
			}).Times(1),
		// Invite the users.
		mockClient.EXPECT().Execute(
			/*req=*/ client.ConversationInvite{
				ChannelId: "newchannelid",
				Users:     []string{"123", "456", "789"},
			},
			/*resp=*/ gomock.AssignableToTypeOf(&client.ChannelResponse{})).DoAndReturn(
			func(req client.ConversationInvite,
				resp *client.ChannelResponse) (string, error) {
				resp.Ok = true
				resp.Channel = client.Channel{Id: "newchannelid", Name: newChannelName()}
				return "raw json", nil
			}).Times(1),
		// Post a welcome message.
		mockClient.EXPECT().Execute(
			/*req=*/ client.PostMessageRequest{
				ChannelId: "newchannelid",
				Text:      "Hello, welcome to today's channel.\nOur new video call link is http://zoom"},
			/*resp=*/ gomock.AssignableToTypeOf(&client.GenericResponse{})).DoAndReturn(
			func(req client.PostMessageRequest, resp *client.GenericResponse) (string, error) {
				resp.Ok = true
				return "raw json", nil
			}).Times(1),
		// Find the old channel, list all the channels until we get the old one.
		mockClient.EXPECT().Execute(
			/*req=*/ client.ChannelListRequest{},
			/*resp=*/ gomock.AssignableToTypeOf(&client.ChannelListResponse{})).DoAndReturn(
			func(req client.ChannelListRequest,
				resp *client.ChannelListResponse) (string, error) {
				resp.Ok = true
				resp.Channels = []client.Channel{
					client.Channel{Id: "x", Name: "general"},
					client.Channel{Id: "y", Name: "covid"},
				}
				resp.Metadata.NextCursor = "cursorX"
				return "raw json", nil
			}).Times(1),
		mockClient.EXPECT().Execute(
			/*req=*/ client.ChannelListRequest{Cursor: "cursorX"},
			/*resp=*/ gomock.AssignableToTypeOf(&client.ChannelListResponse{})).DoAndReturn(
			func(req client.ChannelListRequest,
				resp *client.ChannelListResponse) (string, error) {
				resp.Ok = true
				resp.Channels = []client.Channel{
					client.Channel{Id: "z", Name: "blah"},
					client.Channel{Id: "oldchannelid", Name: oldChannelName()},
					client.Channel{Id: "a", Name: "foochannel"},
				}
				resp.Metadata.NextCursor = "cursorY"
				return "raw json", nil
			}).Times(1),
		// We set a cursor above, but because the channel was found we don't expect
		// another call. Archive the channel.
		mockClient.EXPECT().Execute(
			/*req=*/ client.ChannelArchiveRequest{ChannelId: "oldchannelid"},
			/*resp=*/ gomock.AssignableToTypeOf(&client.GenericResponse{})).DoAndReturn(
			func(req client.ChannelArchiveRequest, resp *client.GenericResponse) (string, error) {
				resp.Ok = true
				return "raw json", nil
			}).Times(1))

	// set the header via
	req, err := http.NewRequest("GET", "/create_channel", nil)
	req.Header.Add("X-Appengine-Cron", "true")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateChannelHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusOK,
		)
	}

	t.Logf("Returned body:\n%v", rr.Body.String())

	t.Logf("Got client: %v", mockClient)
}

func TestPostCall(t *testing.T) {
	mockClient := getClient(t)

	// Set the mocks.

	gomock.InOrder(
		// Create the Call object.
		mockClient.EXPECT().Execute(
			/*req=*/ client.Call{
				ExternalUniqueId:  newChannelName(),
				JoinUrl:           "http://zoom",
				ExternalDisplayId: "123456",
				Title:             "Game Time!",
				StartTimeUnix:     todayAtSixThirty().Unix(),
			},
			/*resp=*/ gomock.AssignableToTypeOf(&client.CallResponse{})).DoAndReturn(
			func(req client.Call,
				resp *client.CallResponse) (string, error) {
				resp.Ok = true
				resp.Call = req
				resp.Call.Id = "987654"
				return "raw json haha", nil
			}).Times(1),
		// Get the channel.
		mockClient.EXPECT().Execute(
			/*req=*/ client.ChannelListRequest{},
			/*resp=*/ gomock.AssignableToTypeOf(&client.ChannelListResponse{})).DoAndReturn(
			func(req client.ChannelListRequest,
				resp *client.ChannelListResponse) (string, error) {
				resp.Ok = true
				resp.Channels = []client.Channel{
					client.Channel{Id: "channelid", Name: newChannelName()},
					client.Channel{Id: "foo", Name: "covid"},
				}
				return "raw json", nil
			}).Times(1),
		// Post a reminder.
		mockClient.EXPECT().Execute(
			/*req=*/ client.PostMessageRequest{
				ChannelId: "channelid",
				Text:      "Join the Video Call",
				Blocks:    []client.Block{client.Block{Type: "call", CallId: "987654"}},
			},
			/*resp=*/ gomock.AssignableToTypeOf(&client.GenericResponse{})).DoAndReturn(
			func(req client.PostMessageRequest, resp *client.GenericResponse) (string, error) {
				resp.Ok = true
				return "raw json", nil
			}).Times(1))

	// set the header via
	req, err := http.NewRequest("GET", "/post_call", nil)
	req.Header.Add("X-Appengine-Cron", "true")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PostCallHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf(
			"unexpected status: got (%v) want (%v)",
			status,
			http.StatusOK,
		)
	}

	t.Logf("Returned body:\n%v", rr.Body.String())
}

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Setenv("VC_URL", "http://zoom")
	os.Setenv("VC_CALL_ID", "123456")
	requireCron = true
	os.Exit(m.Run())
}
