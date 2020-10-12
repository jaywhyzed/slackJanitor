package client_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/jaywhyzed/slackJanitor/client"
	"github.com/jaywhyzed/slackJanitor/client/mocks"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Based on example response from https://api.slack.com/methods/conversations.create
const createChannelResponseSuccess = `
{
    "ok": true,
    "channel": {
        "id": "C0EAQDV4Z",
        "name": "new-channel-name",
        "is_channel": true,
        "is_group": false,
        "is_im": false,
        "created": 1504554479,
        "creator": "U0123456",
        "is_archived": false,
        "is_general": false,
        "unlinked": 0,
        "name_normalized": "endeavor",
        "is_shared": false,
        "is_ext_shared": false,
        "is_org_shared": false,
        "pending_shared": [],
        "is_pending_ext_shared": false,
        "is_member": true,
        "is_private": false,
        "is_mpim": false,
        "last_read": "0000000000.000000",
        "latest": null,
        "unread_count": 0,
        "unread_count_display": 0,
        "topic": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "purpose": {
            "value": "",
            "creator": "",
            "last_set": 0
        },
        "previous_names": [],
        "priority": 0
    }
}`

func getClient(t *testing.T, token string) (*mocks.MockHttpClientInterface, client.Client) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockHttp := mocks.NewMockHttpClientInterface(mockCtrl)
	slackClient := client.NewClientWithHttpClient(mockHttp, token)
	return mockHttp, slackClient
}

// HasToken is a Matcher verifying the auth token is set.
type hasToken struct {
	ExpectedToken string
}

func HasToken(t string) gomock.Matcher {
	return &hasToken{t}
}

func (ht hasToken) String() string {
	return "Checks whether the object has the expected token '" + ht.ExpectedToken + "'"
}

func (ht hasToken) Matches(x interface{}) bool {
	authHeaders := x.(*http.Request).Header["Authorization"]
	return len(authHeaders) == 1 && authHeaders[0] == "Bearer "+ht.ExpectedToken
}

// HasUrl is a Matcher verifying the URL of the request.
type hasUrl struct {
	URL string
}

func HasUrl(url string) gomock.Matcher {
	return &hasUrl{url}
}

func (hu hasUrl) Matches(x interface{}) bool {
	return x.(*http.Request).URL.String() == hu.URL
}

func (hu hasUrl) String() string {
	return "Check whether the request has the URL " + hu.URL
}

// HasJsonBody is a Matcher verifying the body content
type hasJsonBody struct {
	JsonContent string
}

func HasJsonBody(json string) gomock.Matcher {
	return &hasJsonBody{json}
}

func (hb hasJsonBody) Matches(x interface{}) bool {

	ioReadCloser := x.(*http.Request).Body
	buf := new(bytes.Buffer)
	buf.ReadFrom(ioReadCloser)
	log.Printf("Body is\n%s", buf.String())

	var actual, expected interface{}
	err := json.Unmarshal(buf.Bytes(), &actual)
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}
	actualMap := actual.(map[string]interface{})

	err = json.Unmarshal(bytes.NewBufferString(hb.JsonContent).Bytes(), &expected)
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}
	expectedMap := expected.(map[string]interface{})

	return reflect.DeepEqual(expectedMap, actualMap)
}

func (hb hasJsonBody) String() string {
	return "Object has JSON body equivalent to " + hb.JsonContent
}

func HttpResponseWithBody(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
	}
}

func ExpectEqual(t *testing.T, slackClient client.Client, req client.Request,
	actual, expected interface{}) {

	raw_resp, err := slackClient.Execute(req, actual)
	if err != nil {
		t.Errorf("Got error: %s", err.Error())
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected:\n%+v\n\nbut got:\n%+v\n\nUnparsed response was\n%s",
			expected, actual, raw_resp)
	}
}

// Test a successful CreateChannel request.
func TestCreateChannel(t *testing.T) {
	mockHttp, slackClient := getClient(t, "my-auth-token")

	mockHttp.EXPECT().Do(gomock.All(
		HasToken("my-auth-token"),
		HasUrl("https://slack.com/api/conversations.create"),
		HasJsonBody(`{"name": "new-channel-name"}`))).Return(
		HttpResponseWithBody(createChannelResponseSuccess), nil).Times(1)

	var actual client.ChannelResponse
	ExpectEqual(t, slackClient,
		/*req=*/ client.CreateChannelRequest{Name: "new-channel-name"},
		/*actual=*/ &actual,
		/*expected=*/ &client.ChannelResponse{
			Ok:      true,
			Channel: client.Channel{Id: "C0EAQDV4Z", Name: "new-channel-name"}})
}

// Test a non-200 error.
func TestCreateChannel404(t *testing.T) {
	mockHttp, slackClient := getClient(t, "my-auth-token")

	mockHttp.EXPECT().Do(gomock.All(
		HasToken("my-auth-token"),
		HasUrl("https://slack.com/api/conversations.create"),
		HasJsonBody(`{"name": "new-channel-name"}`))).Return(
		&http.Response{
			Status:     "404 NOT FOUND",
			StatusCode: 404,
			Proto:      "HTTP/1.0",
			Body:       ioutil.NopCloser(bytes.NewBufferString("error\r\n")),
		}, nil).Times(1)

	var resp client.ChannelResponse
	_, err := slackClient.Execute(
		/*req=*/ client.CreateChannelRequest{Name: "new-channel-name"},
		/*resp=*/ &resp)

	if err == nil {
		t.Errorf("Failed to get an error!")
	} else if strings.Index(err.Error(), "non-200 response") == -1 {
		log.Printf("JZ Got unexpected error:\n%s", err)
	}
}

// Test a network level error.
func TestTransportError(t *testing.T) {
	mockHttp, slackClient := getClient(t, "my-auth-token")

	mockHttp.EXPECT().Do(gomock.All(
		HasToken("my-auth-token"),
		HasUrl("https://slack.com/api/conversations.create"),
		HasJsonBody(`{"name": "new-channel-name"}`))).Return(
		nil, errors.New("Some network error happened")).Times(1)

	var resp client.ChannelResponse
	_, err := slackClient.Execute(
		/*req=*/ client.CreateChannelRequest{Name: "new-channel-name"},
		/*resp=*/ &resp)

	if err == nil {
		t.Errorf("Failed to get an error!")
	} else if err.Error() != "Some network error happened" {
		log.Printf("JZ Got unexpected error:\n%s", err)
	}
}

// Successful UsersListRequest (which uses GET, not POST).
func TestUsersListRequest(t *testing.T) {
	mockHttp, slackClient := getClient(t, "my-auth-token")

	mockHttp.EXPECT().Do(gomock.All(
		HasToken("my-auth-token"),
		HasUrl("https://slack.com/api/users.list"),
		gomock.Any())).Return(
		HttpResponseWithBody(`
{
  "ok": true,
  "members": [
    { "id": "123", "name": "foo" },
    { "id": "456", "name": "bar" }
  ],
  "response_metadata": {
    "next_cursor": "XXXcontinuationXXX"
  }
}
`), nil).Times(1)

	mockHttp.EXPECT().Do(gomock.All(
		HasToken("my-auth-token"),
		HasUrl("https://slack.com/api/users.list?cursor=XXXcontinuationXXX"),
		gomock.Any())).Return(
		HttpResponseWithBody(`
{
  "ok": true,
  "members": [
    { "id": "789", "name": "baz" }
  ]
}
`), nil).Times(1)

	{
		var actual client.UsersListResponse
		ExpectEqual(t, slackClient,
			/*req=*/ client.UsersListRequest{},
			/*actual=*/ &actual,
			/*expected=*/ &client.UsersListResponse{
				Ok: true,
				Members: []client.User{
					client.User{Id: "123", Name: "foo"},
					client.User{Id: "456", Name: "bar"},
				},
				Metadata: client.ResponseMetadata{
					NextCursor: "XXXcontinuationXXX",
				},
			})
	}

	{
		var actual client.UsersListResponse
		ExpectEqual(t, slackClient,
			/*req=*/ client.UsersListRequest{Cursor: "XXXcontinuationXXX"},
			/*actual=*/ &actual,
			/*expected=*/ &client.UsersListResponse{
				Ok: true,
				Members: []client.User{
					client.User{Id: "789", Name: "baz"},
				},
			})
	}
}
