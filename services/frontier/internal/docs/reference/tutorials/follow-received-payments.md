---
title: Follow Received Payments
---

This tutorial shows how easy it is to use Frontier to watch for incoming payments on an [account](../resources/account.md)
using JavaScript and `EventSource`.  We will eschew using [`xdb-digitalbits-sdk`](https://github.com/xdbfoundation/xdb-digitalbits-sdk), the
high-level helper library, to show that it is possible for you to perform this
task on your own, with whatever programming language you would like to use.

This tutorial assumes that you:

- Have node.js installed locally on your machine. (Node version >= 12.0.0)
- Have curl installed locally on your machine.
- Are running on Linux, OS X, or any other system that has access to a bash-like
  shell.
- Are familiar with launching and running commands in a terminal.

In this tutorial we will learn:

- How to create a new account.
- How to fund your account using friendbot.
- How to follow payments to your account using curl and EventSource.

## Project Skeleton

Let's get started by building our project skeleton:

```bash
$ npm init
$ npm install xdb-digitalbits-base --save
```

You can check that everything went well by running the following command:

```bash
$ node -e "require('xdb-digitalbits-base')"
```

Everything was successful if no output it generated from the above command.  Now
let's write a script to create a new account.

## Creating an account

Create a new file named `make_account.js` and paste the following text into it:

```javascript
var Keypair = require("xdb-digitalbits-base").Keypair;

var newAccount = Keypair.random();

console.log("New key pair created!");
console.log("  Account ID: " + newAccount.publicKey());
console.log("  Secret: " + newAccount.secret());
```

Save the file and run it:

```bash
$ node make_account.js
New key pair created!
  Account ID: GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT
  Secret: SBODKDHUKBCXEPDZZBANAPKV2BNH32RWL2PY6OYIUH7FQZ3L2XYVPUJU
$
```

Before our account can do anything it must be funded.  Indeed, before an account
is funded it does not truly exist!

## Funding your account

The DigitalBits test network provides the Friendbot, a tool that developers
can use to get testnet digitalbits for testing purposes. To fund your account, simply
execute the following curl command:

```bash
$ curl "https://friendbot.testnet.digitalbits.io/?addr=GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT"
```

Don't forget to replace the account id above with your own.  If the request
succeeds, you should see a response like:

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2"
    },
    "account": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM"
    },
    "ledger": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/933127"
    },
    "operations": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2/operations{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2/effects{?cursor,limit,order}",
      "templated": true
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc\u0026cursor=4007749948018688"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc\u0026cursor=4007749948018688"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2"
    }
  },
  "id": "26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2",
  "paging_token": "4007749948018688",
  "successful": true,
  "hash": "26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2",
  "ledger": 933127,
  "created_at": "2021-06-14T17:53:02Z",
  "source_account": "GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM",
  "source_account_sequence": "1099511627777",
  "fee_account": "GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM",
  "fee_charged": "300",
  "max_fee": "300",
  "operation_count": 1,
  "envelope_xdr": "AAAAAgAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAASwAAAEAAAAAAQAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAALYlko2FbY34B5mNfTQSA84/EDC5PbwfQdvACSxCQbhFAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIdugAAAAAAAAAAAICQ30+AAAAQCmdeGqUTvsUeRFjQABMVyRFL2IyUKzIDrsRXyXA29sYmTQwSwvlof7MmghCqyUqmzHfzHdgiPSyuK+17T/LswpCQbhFAAAAQBEbA/9+Nh/sR6YixIAQM2sBibkvFOu4U9W5h13dWi/NFMPihDshyv4MNBgfXVI0A3pglNiShaBkgxWikVPGigU=",
  "result_xdr": "AAAAAAAAASwAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
  "result_meta_xdr": "AAAAAgAAAAIAAAADAA49BwAAAAAAAAAAgL8fauI9jNTFjdCwNjAHs0Sf/NACSMuxBz/BoAJDfT4AAAAAPDNfVAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA49BwAAAAAAAAAAgL8fauI9jNTFjdCwNjAHs0Sf/NACSMuxBz/BoAJDfT4AAAAAPDNfVAAAAQAAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMADYQDAAAAAAAAAAC2JZKNhW2N+AeZjX00EgPOPxAwuT28H0HbwAksQkG4RQLGiHdubrLAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAAC2JZKNhW2N+AeZjX00EgPOPxAwuT28H0HbwAksQkG4RQLGiGAl98rAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAADj0HAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIdugAAA49BwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAA=",
  "fee_meta_xdr": "AAAABAAAAAMAAAEAAAAAAAAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAAAA8M2CAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAAAA8M19UAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADYQDAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxY8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxdoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
  "memo_type": "none",
  "signatures": [
    "KZ14apRO+xR5EWNAAExXJEUvYjJQrMgOuxFfJcDb2xiZNDBLC+Wh/syaCEKrJSqbMd/Md2CI9LK4r7XtP8uzCg==",
    "ERsD/342H+xHpiLEgBAzawGJuS8U67hT1bmHXd1aL80Uw+KEOyHK/gw0GB9dUjQDemCU2JKFoGSDFaKRU8aKBQ=="
  ],
  "valid_after": "1970-01-01T00:00:00Z"
}
```

After a few seconds, the DigitalBits network will perform consensus, close the
ledger, and your account will have been created.  Next up we will write a command
that watches for new payments to your account and outputs a message to the
terminal.

## Following payments using `curl`

To follow new payments connected to your account you simply need to send `Accept: text/event-stream` header to the [/payments](../endpoints/payments-all.md) endpoint.

```bash
$ curl -H "Accept: text/event-stream" "https://frontier.testnet.digitalbits.io/accounts/GB7JFK56QXQ4DVJRNPDBXABNG3IVKIXWWJJRJICHRU22Z5R5PI65GAK3/payments"
```

As a result you will see something like:

```bash
retry: 1000
event: open
data: "hello"

id: 4007749948018689
data: {
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4007749948018689"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4007749948018689/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4007749948018689"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4007749948018689"
    }
  },
  "id": "4007749948018689",
  "paging_token": "4007749948018689",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-06-14T17:53:02Z",
  "transaction_hash": "26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2",
  "starting_balance": "10000.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT"
}
```

Every time you receive a new payment you will get a new row of data. Payments is not the only endpoint that supports streaming. You can also stream transactions [/transactions](../endpoints/transactions-all.md) and operations [/operations](../endpoints/operations-all.md).

## Following payments using `EventStream`

> **Warning!** `EventSource` object does not reconnect for certain error types so it can stop working.
> If you need a reliable streaming connection please use our [SDK](https://github.com/xdbfoundation/xdb-digitalbits-sdk).

Another way to follow payments is writing a simple JS script that will stream payments and print them to console. Create `stream_payments.js` file and paste the following code into it:

```js
var EventSource = require('eventsource');
var es = new EventSource('https://frontier.testnet.digitalbits.io/accounts/GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT/payments');
es.onmessage = function(message) {
	var result = message.data ? JSON.parse(message.data) : message;
	console.log('New payment:');
	console.log(result);
};
es.onerror = function(error) {
	console.log('An error occurred!');
}
```

Now, run our script: `node stream_payments.js`. You should see following output:

```bash
New payment:
{ _links:
   { self:
      { href:
         'https://frontier.testnet.digitalbits.io/operations/4007749948018689' },
     transaction:
      { href:
         'https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2' },
     effects:
      { href:
         'https://frontier.testnet.digitalbits.io/operations/4007749948018689/effects' },
     succeeds:
      { href:
         'https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4007749948018689' },
     precedes:
      { href:
         'https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4007749948018689' } },
  id: '4007749948018689',
  paging_token: '4007749948018689',
  transaction_successful: true,
  source_account: 'GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP',
  type: 'create_account',
  type_i: 0,
  created_at: '2021-06-14T17:53:02Z',
  transaction_hash:
   '26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2',
  starting_balance: '10000.0000000',
  funder: 'GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP',
  account: 'GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT' }
```

## Testing it out

We now know how to get a stream of transactions to an account. Let's check if our solution actually works and if new payments appear. Let's watch as we send a payment ([`create_account` operation](https://developers.digitalbits.io/guides/concepts/list-of-operations.html#create-account)) from our account to another account.

We use the `create_account` operation because we are sending payment to a new, unfunded account. If we were sending payment to an account that is already funded, we would use the [`payment` operation](https://developers.digitalbits.io/guides/concepts/list-of-operations.html#payment).

First, let's check our account sequence number so we can create a payment transaction. To do this we send a request to frontier:

```bash
$ curl "https://frontier.testnet.digitalbits.io/accounts/GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT"
```

Sequence number can be found under the `sequence` field. The current sequence number is `4007749948014592`. Save this value somewhere.

Now, create `make_payment.js` file and paste the following code into it:

```js
var DigitalBitsBase = require("xdb-digitalbits-base");

var keypair = DigitalBitsBase.Keypair.fromSecret('SBODKDHUKBCXEPDZZBANAPKV2BNH32RWL2PY6OYIUH7FQZ3L2XYVPUJU');
var account = new DigitalBitsBase.Account(keypair.publicKey(), "4007749948014592");

var amount = "1000";
var transaction = new DigitalBitsBase.TransactionBuilder(account, {
    networkPassphrase: DigitalBitsBase.Networks.TESTNET,
    fee: DigitalBitsBase.BASE_FEE,
})
  .addOperation(DigitalBitsBase.Operation.createAccount({
    destination: DigitalBitsBase.Keypair.random().publicKey(),
    startingBalance: amount
  }))
  .setTimeout(180)
  .build();

transaction.sign(keypair);

console.log(transaction.toEnvelope().toXDR().toString("base64"));
```

After running this script you should see a signed transaction blob. To submit this transaction we send it to frontier or digitalbits-core. But before we do, let's open a new console and start our previous script by `node stream_payments.js`.

Now to send a transaction just use frontier:

```bash
curl -H "Content-Type: application/json" -X POST -d '{"tx":"AAAAAgAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAAGQADj0HAAAAAQAAAAEAAAAAAAAAAAAAAABgx6LlAAAAAAAAAAEAAAAAAAAAAAAAAACPXpmxADuBw6GNwLxN65XPknH64rk9L94e9z+mDNuTywAAAAJUC+QAAAAAAAAAAAGMuWhbAAAAQAlkp9gX0/JHnfAQSTlJ8DnrL69fBgJslx5Fjt7Fc0io2DSnFKVUKRjM2sZi9bIqp7V6L59BZbcbhDI1DkdFAgs="}' "https://frontier.testnet.digitalbits.io/transactions"
```

You should see a new payment in a window running `stream_payments.js` script.
