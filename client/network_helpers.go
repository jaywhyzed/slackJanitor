package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Populates resp, returns the raw json string and an error.
func Execute(req Request, resp interface{}) (string, error) {
	http_req, err := newRequest(req)
	if err != nil {
		return "", err
	}
	return executeHttpReq(http_req, resp)
}

func ExecuteOrDie(req Request, resp interface{}) string {
	raw_string, err := Execute(req, &resp)
	if err != nil {
		log.Fatalf("Fatal error executing request! %v\n\nRequest:\n%+v",
			err, req)
	}
	return raw_string
}

func botUserToken() string {
	token := os.Getenv("SLACK_BOT_USER_TOKEN")
	if len(token) == 0 {
		log.Fatal("Missing SLACK_BOT_USER_TOKEN!")
	}
	return token
}

func newRequest(body Request) (*http.Request, error) {
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
	req.Header.Add("Authorization", "Bearer "+botUserToken())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return req, nil
}

func executeHttpReq(req *http.Request, resp interface{}) (string, error) {
	client := http.Client{}
	http_resp, err := client.Do(req)

	if err != nil {
		log.Printf("Error while executing request:\n%+v\n\nError is:\n%v\n", *req, err)
		return "", errors.New("Error executing request: " + err.Error())
	}
	defer http_resp.Body.Close()

	raw_resp, err := ioutil.ReadAll(http_resp.Body)
	if err != nil {
		log.Printf("Error while reading response:\n%v", err)
		return "", errors.New("Error reading response: " + err.Error())
	}
	err = json.Unmarshal(raw_resp, &resp)
	if err != nil {
		log.Printf("Error while unmarshaling response, response was:\n%v\n\nError:\n%v", string(raw_resp), err)
		return "", errors.New("Error unmarshaling response: " + err.Error())
	}

	return string(raw_resp), nil
}
