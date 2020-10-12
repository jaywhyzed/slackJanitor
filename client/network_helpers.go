package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
)

type Request interface {
	URL() string
	Verb() string
}

//go:generate mockgen --destination=mocks/mock_client.go --package mocks . Client
type Client interface {
	// Execute makes an HTTP request with the given Request,
	// and populates resp with the JSON response.
	// Returns the raw response text, and an error (nil on success).
	Execute(req Request, resp interface{}) (string, error)
}

// HTTPClient interface, this is implemented by http.Client
//go:generate mockgen --destination=mocks/mock_http_client.go --package mocks . HttpClientInterface
type HttpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type ClientImpl struct {
	httpClient HttpClientInterface
	token      string
}

func NewClientWithHttpClient(httpClient HttpClientInterface, token string) Client {
	return &ClientImpl{
		httpClient: httpClient,
		token:      token,
	}
}

func NewClient(token string) Client {
	return NewClientWithHttpClient(&http.Client{}, token)
}

// Populates resp, returns the raw json string and an error.
func (c ClientImpl) Execute(req Request, resp interface{}) (string, error) {
	// Reset the resp pointer in case it's not empty.
	p := reflect.ValueOf(resp).Elem()
	p.Set(reflect.Zero(p.Type()))

	http_req, err := newRequest(req, c.token)
	if err != nil {
		return "", err
	}
	log.Printf("Calling URL %s", req.URL())
	json, err := executeHttpReq(c.httpClient, http_req, resp)
	if err != nil {
		log.Printf("Got error making HTTP request: %v", err)
	} else {
		log.Printf("Got response:\n%+v", resp)
	}

	return json, err
}

func newRequest(body Request, bearerToken string) (*http.Request, error) {
	buf := new(bytes.Buffer)
	if body.Verb() == "POST" {
		log.Printf("Creating POST request with body:\n%v", body)
		json.NewEncoder(buf).Encode(body)
	}
	log.Printf("Creating request with URL %s", body.URL())
	req, err := http.NewRequest(body.Verb(), body.URL(), buf)
	if err != nil {
		return nil, errors.New("Error creating http.Request: " + err.Error())
	}
	req.Header.Add("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return req, nil
}

func executeHttpReq(httpClient HttpClientInterface, req *http.Request, resp interface{}) (string, error) {
	http_resp, err := httpClient.Do(req)

	if err != nil {
		return "", errors.New("Error executing request: " + err.Error())
	}
	defer http_resp.Body.Close()

	if http_resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("Got non-200 response:\n %+v", http_resp))
	}

	raw_resp, err := ioutil.ReadAll(http_resp.Body)
	if err != nil {
		return "", errors.New("Error reading response: " + err.Error())
	}
	err = json.Unmarshal(raw_resp, &resp)
	if err != nil {
		return string(raw_resp), errors.New(
			fmt.Sprintf("Error unmarshaling response: %s", err.Error()))
	}

	return string(raw_resp), nil
}
