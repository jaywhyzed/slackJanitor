package janitor

// TODO: Move all this to main/cli, See https://github.com/GoogleCloudPlatform/functions-framework-go#quickstart-hello-world-on-your-local-machine
// package main

// import (
// 	"log"
// 	"net/http"
// 	"os"

// 	"github.com/jaywhyzed/slackJanitor"
// )

// func main() {

// 	http.HandleFunc("/", janitor.IndexHandler)
// 	http.HandleFunc("/create_channel", janitor.CreateChannelHandler)
// 	http.HandleFunc("/post_call", janitor.PostCallHandler)

// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "8080"
// 		log.Printf("Defaulting to port %s", port)
// 	}

// 	log.Printf("Listening on port %s", port)
// 	if err := http.ListenAndServe(":"+port, nil); err != nil {
// 		log.Fatal(err)
// 	}
// }
