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

func main() {
	// Friendbot must be running
	// Get Friendbot URL from CL. Friendbot must be running as a local server.
	fbURL := flag.String("url", "http://0.0.0.0:8000/", "URL of friendbot")
	numRequests := flag.Int("requests", 500, "number of requests")
	flag.Parse()
	durations := []time.Duration{}
	for i := 0; i < 500; i++ {
		kp, err := keypair.Random()
		if err != nil {
			panic(err)
		}
		address := kp.Address()
		err = makeFriendbotRequest(address, *fbURL, &durations)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	log.Printf("Got %d times with average %d", *numRequests, mean(durations))
}

func makeFriendbotRequest(address, fbURL string, durations *[]time.Duration) error {
	defer timeTrack(time.Now(), "makeFriendbotRequest", durations)
	formData := url.Values{
		"addr": {address},
	}
	resp, err := http.PostForm(fbURL, formData)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	log.Print(result)
	return nil
}

func timeTrack(start time.Time, name string, durations *[]time.Duration) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
	*durations = append(*durations, elapsed)
}

func mean(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}
