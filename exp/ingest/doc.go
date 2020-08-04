/*

Package ingest provides primitives for building custom ingestion engines.

Very often developers need features that are outside of Horizon's scope. While it
provides APIs for building the  most common apps, it's not possible to add all
possible features. This is why this package was created.

Ledger Backend

Ledger backends are sources of information about Stellar network ledgers. This
can be either Stellar-Core DB, Captive Stellar-Core or History Archives.
Please consult the ledgerbackend package docs for more information about each
backend.

Warning: Please note that ledger backends provide low-level xdr.LedgerCloseMeta
that should not be used directly unless the developer really understands this data
structure. Read on to understand how to use ledger backend in higher level objects.

Readers

Readers are objects that wrap ledger backend and provide higher level, developer
friendly APIs for reading ledger data.

Currently there are three types of readers (all in ingest/io package):
  * SingleLedgerStateReader reads ledger entries from history buckets for a
    given checkpoint ledger. Allow building state (all accounts, trust lines
    etc.) at any checkpoint ledger.
  * LedgerTransactionReader reads transactions for a given ledger sequence.
  * LedgerChangeReader reads all changes to ledger entries created as a result
    of transactions (fees and meta) and protocol upgrades in a given ledger.

Warning: Please note that readers stream both successful and failed transactions.
Please check transactions status in your application (if required).

Processors

Processors allow building pipelines for ledger processing. This allows better
separation of concerns in ingestion engines: some processors can be responsible
for trading operations, other for payments, etc. This also allows easier
configurations of pipelines. Some features can be turned off by simply disabling
a single processor.

There are two types of processors (ingest/io package):
  * ChangeProcessor responsible for processing changes to ledger entries
    (io.Change).
  * TransactionProcessor reponsible for processing transactions
    (io.LedgerTransaction).

For an object to be a processor, it needs to implement a single method:
ProcessChange or ProcessTransaction. This is a very simple yet powerful interface
that allows building many kinds of features.

*/
package ingest
