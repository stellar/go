## Submitter

A WIP project for submitting Stellar transactions to the network with high throughput.

### Testing

#### Create the database

```sh
$ psql
```

```sql
> CREATE DATABASE submitter;
> \c submitter
> CREATE TABLE transactions (
    id int NOT NULL,
    external_id varchar(256) NOT NULL,
    state varchar(256) NOT NULL,
    sending_at timestamp,
    sent_at timestamp,
    destination varchar(256) NOT NULL,
    amount varchar(256) NOT NULL,
    hash varchar(256)
);
> INSERT INTO transactions (
    id, 
    external_id, 
    state, 
    destination, 
    amount
) VALUES (
    1, 
    '1', 
    'pending', 
    'GCMN2TNLYZ4AQ46LBRV4OKNKM6K4S4Z46AEYHUDUOHGBXZAIIAIHUC6N', 
    '10'
);
```

#### Run it

```sh
export SUBMITTER_NUM_CHANNELS=
export SUBMITTER_ROOT_SEED=
export SUBMITTER_MAX_BASE_FEE=
go run ./main.go
```

This should successfully process the payment via a derived channel account and save the transaction hash to the DB.