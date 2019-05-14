package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/stellar/go/keypair"
)

type maybeDuration struct {
	maybeDuration time.Duration
	maybeError    error
}

func main() {
	// Friendbot must be running
	// Get Friendbot URL from CL. Friendbot must be running as a local server.
	fbURL := flag.String("url", "http://0.0.0.0:8000/", "URL of friendbot")
	numRequests := flag.Int("requests", 500, "number of requests")
	flag.Parse()
	durationChannel := make(chan time.Duration, *numRequests)
	for i := 0; i < *numRequests; i++ {
		kp, err := keypair.Random()
		if err != nil {
			panic(err)
		}
		address := kp.Address()
		go makeFriendbotRequest(address, *fbURL, durationChannel)

		time.Sleep(time.Duration(500) * time.Millisecond)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Duration(10) * time.Second)
	durations := []time.Duration{}
	for i := 0; i < *numRequests; i++ {
		durations = append(durations, <-durationChannel)
	}
	close(durationChannel)
	log.Printf("Got %d times with average %s", *numRequests, mean(durations))
}

func makeFriendbotRequest(address, fbURL string, durationChannel chan time.Duration) {
	defer timeTrack(time.Now(), "makeFriendbotRequest", durationChannel)
	formData := url.Values{
		"addr": {address},
	}
	resp, _ := http.PostForm(fbURL, formData)
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	log.Print(result)
}

func timeTrack(start time.Time, name string, durationChannel chan time.Duration) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
	durationChannel <- elapsed
}

func mean(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}
