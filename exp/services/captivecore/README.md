# captivecore

The Captive DigitalBits-Core Server allows you to run a dedicated DigitalBits-Core instance
for the purpose of ingestion. The server must be bundled with a DigitalBits Core binary.

If you run Frontier with Captive DigitalBits-Core ingestion enabled Frontier will spawn a DigitalBits-Core
subprocess. Frontier's ingestion system will then stream ledgers from the subprocess via
a filesystem pipe. The disadvantage of running both Frontier and the DigitalBits-Core subprocess
on the same machine is it requires detailed per-process monitoring to be able to attribute
potential issues (like memory leaks) to a specific service.

Now you can run Frontier and pair it with a remote Captive DigitalBits-Core instance. The
Captive DigitalBits-Core Server can run on a separate machine from Frontier. The server
will manage DigitalBits-Core as a subprocess and provide an HTTP API which Frontier
can use remotely to stream ledgers for the purpose of ingestion.

Note that, currently, a single Captive DigitalBits-Core Server cannot be shared by
multiple Frontier instances.

## API

### `GET /latest-sequence`

Fetches the latest ledger sequence available on the captive core instance.

Response:

```json
{
	"sequence": 12345
}
```


### `GET /ledger/<sequence>`

Fetches the ledger with the given sequence number from the captive core instance.

Response:


```json
{
    "present": true,
    "ledger": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
}
```

### `POST /prepare-range`

Preloads the given range of ledgers in the captive core instance.

Bounded request:
```json
{
    "from": 123,
    "to":   150,
    "bounded": true
}
```

Unbounded request:
```json
{
    "from": 123,
    "bounded": false
}
```

Response:
```json
{
    "ledgerRange": {"from":  123, "bounded":  false},
    "startTime": "2020-08-31T13:29:09Z",
    "ready": true,
    "readyDuration": 1000
}
```

## Usage

```
$ captivecore --help
Run the Captive DigitalBits-Core Server

Usage:
  captivecore [flags]

Flags:
      --db-url                             Frontier Postgres URL (optional) used to lookup the ledger hash for sequence numbers
      --digitalbits-core-binary-path           Path to digitalbits core binary
      --digitalbits-core-config-path           Path to digitalbits core config file
      --history-archive-urls               Comma-separated list of digitalbits history archives to connect with
      --log-level                          Minimum log severity (debug, info, warn, error) to log (default info)
      --network-passphrase string          Network passphrase of the DigitalBits network transactions should be signed for (NETWORK_PASSPHRASE) (default "TestNet Global DigitalBits Network ; December 2020")
      --port int                           Port to listen and serve on (PORT) (default 8000)
```