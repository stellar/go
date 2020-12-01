# Ingestion Finite State Machine
The following states are possible:
  - `start`
  - `stop`
  - `build`
  - `resume`
  - `stressTest`
  - `verifyRange`
  - `historyRange`
  - `reingestHistoryRange`
  - `waitForCheckpoint`

#### Definitions
There are some important terms that need to be defined for clarity as they're used extensively in both the codebase and this breakdown:

  - the `historyQ` field corresponds to **historical *time-series* data**, *not* to the history *archives* (which only contain *cumulative* data); I prefer to refer to it as the `TimeSeriesDB`.
  - the `lastIngestedLedger` thus corresponds to the last ledger that Horizon  ingested into the time-series tables (coming from the `key_value_store` table)
  - the `lastHistoryLedger`, however, corresponds to the last ledger that Core+Horizon are *aware* of, not necessarily *ingested* (coming from `history_ledgers` table); I'll usually refer to it as the `lastKnownLedger`
  - the `lastCheckpoint` corresponds to the last checkpoint ledger (reminder: a checkpoint ledger is one in which: `(ledger# + 1) mod 64 == 0`) and thus to a matching history archive upload.

One of the most important jobs of the FSM described here is to make sure that `lastIngestedLedger` and `lastHistoryLedger` are equal: the [`historyRange`](#historyrange-state) updates the latter, but not the former, so that we can track when state data is behind history data.

In general, only one node should ever be writing to a database at once, globally. Hence, there are a few checks at the start of most states to ensure this.


#### Tables
Within the Horizon database, there are a number of tables touched by ingestion that are worth differentiating explicitly. With these in mind, the subsequently-described states and their respective operations should be much clearer.

The database contains:

  - **History tables** -- all tables that contain historical time-series data, such as `history_transactions`, `history_operations`, etc.
  - **Transaction processors** -- processors that ingest the history tables (described by the `io.LedgerTransaction` interface).
  - **State tables** -- all tables that contain the current cumulative state, such as accounts, offers, etc.
  - **Change processors** -- processors ingesting tx meta (time-series data) that update state tables. These aren't related to a particular *transaction*, but rather describe a *transition* of a ledger entry from one state to another (described by the `io.Change` interface).


## `start` State 
As you might expect, this state kicks off the FSM process.

There are a few possible branches in this state.

##### DB upgrade or fresh start
The "happiest path" is the following: either the ingestion database is empty, so we can start purely from scratch, or the state data in a database is outdated, meaning it needs to be upgraded and so can effectively be started from scratch after catch-up.

This branches differently depending on the last known ledger:

  - If it's newer than the last checkpoint, we need to wait for a new checkpoint to get the latest cumulative data. Note that though we probably *could* make incremental changes from block to block to the cumulative data, that would be more effort than it's worth relative to just waiting on the next history archive to get dumped. **Next state**: [`waitForCheckpoint`](#waitforcheckpoint-state).

  - If it's older, however, then we can just grok the missing gap (i.e. until the *latest* checkpoint) and build up (only) the time-series data. **Next state**: [`historyRange`](#historyrange-state).

In the other cases (matching last-known and last-checkpoint ledger, or no last-known), **next state**: [`build`](#build-state).

##### Otherwise
If we can't have a clean slate to work with, we need to fix partial state. Specifically,

  - If the last-known ledger is ahead of the last-ingested ledger, then Horizon's cumulative state data is behind its historical time-series data in the database. Here, we'll reset the time-series DB and start over. **Next state**: [`start`](#start-state), with `lastIngestedLedger == 0`.

  - If the time-series database is newer than the last-known ledger (can occur if ingestion was done for a different range earlier, for example), then Horizon needs to become aware of the missing ledgers. **Next state**: [`historyRange`](#historyrange-state) from "last known" to "last stored" in time-series db.

**Next state**: [`resume`](#resume-state)


## `build` state
This is the big kahuna of the state machine: there aren't many state transitions aside from success/failure, and all roads ultimately should lead to ~~Rome~~ `build` in order to get ingestion done. This state only establishes a baseline for the cumulative data, though.


### Properties
This state tracks the:

  - `checkpointLedger`, which is Horizon's last-known (though possibly-unprocessed) checkpoint ledger, and 
  - `stop`, which optionally (though *universally*) transitions to the [`stop`](#stop-state) after this state is complete.

### Process
If any of the checks (incl. the aforementioned sync checks) fail, we'll move to the [`start` state](#start-state). Sometimes, though, we want to [`stop`](#stop-state), instead (see `buildState.stop`).

The actual ingestion involves a few steps:

   - turn a checkpoint's history archive into cumulative db data
   - update the ingestion database's version 
   - update the last-ingested ledger in the time-series db
   - commit to the ingestion db

These are detailed later, in the [Ingestion](#ingestion) section. Suffice it to say that at the end of this state, either we've errored out (described above), stopped (on error **or** success, if `buildState.stop` is set), or [`resume`](#resume-state)d from the checkpoint ledger.


## `resume` state
This state ingests time-series data for a single ledger range, then loops back to itself for the next ledger range.


### Properties
This state just tracks one thing:

  - `latestSuccessfullyProcessedLedger`, whose name should be self-explanatory: this indicates the highest ledger number to be ingested.

### Process
First, note the difference between `resumeState.latestSuccessfullyProcessedLedger` and the queried `lastIngestedLedger`: one of these is tied to the state machine, while the other is associated with the actual time-series database. 

The following are problematic error conditions:

  - the former is larger than the latter
  - the versions (of the DB and current ledgers) mismatch
  - the last-known ledger of the time-series db doesn't match the last-ingested ledger
  - **Next state**: [`start`](#start-state).

Otherwise, we have `ingestLedger == lastIngestedLedger + 1`, and will proceed to process the range from `ingestLedger` onward.

With the range prepared, only one other primary state transition is possible. If the last-known ledger of the Core backend is outdated relative to the above `ingestLedger`, we'll block until the requisite ledger is seen and processed by Core. **Next state**: [`resume`](#resume-state) again, with the last-processed ledger set to whatever is last-known to Core.

Otherwise, we can actually turn the ledger into time-series data: this is exactly the responsibility of `RunAllProcessorsOnLedger` and all of its subsequent friends. The deltas for the ledger(s) are ingested into the time-series db, then verified.

**Next state**: [`resume`](#resume-state) again, except now targeting the *next* ledger.


## `historyRange` state
The purpose of this state is to ingest a particular ledger range into the cumulative database. Since the next state will be [start](#start-state), we will be rebuilding state in the future anyway.

### Properties
This tracks an inclusive ledger range: [`fromLedger`, `toLedger`].

**Next state**: [`start`](#start-state)


## `reingestHistoryRange` state
This state acts much like the [`historyRange` state](#historyrange-state).

**Next state**: [`stop`](#stop-state)

### Properties
This tracks an inclusive ledger range: [`fromLedger`, `toLedger`], as well as a `force` flag that will override certain restrictions.


## `waitForCheckpoint` state
This pauses the state machine for 10 seconds then tries again, in hopes that a new checkpoint ledger has been created (remember, checkpoints occur every 64 ledgers).

**Next state**: [`start`](#start-state)


# Ingestion
TODO

# Range Preparation
TODO: See `maybePrepareRange`
