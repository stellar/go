# Stellar Market Tracker

The Stellar Market Tracker allows you to monitor the spreads of any desired asset pairs, and makes them available for Prometheus to scrape.

## Running the project

This project was built using Go 1.13 and [Go Modules](https://blog.golang.org/using-go-modules)

1. Clone the repo and `cd` into it
2. Configure the assets pairs you want to monitor, as well as the refresh interval through `config.json`
3. Build the project: `go build .`
4. Run the project `./market-tracker`
5. Open `http://127.0.01:2112/metrics` and you should be able to view the metrics. This is the endpoint Prometheus should scrape

## Roadmap
- [x] Monitor spreads
- [ ] Monitor market sizes