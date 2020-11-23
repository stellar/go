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
  - the `lastIngestedLedger` thus corresponds to the last ledger that Horizon  ingested into the time-series tables
  - the `lastHistoryLedger`, however, corresponds to the last ledger that Core+Horizon are *aware* of, not necessarily *ingested*; I refer to it as the `lastKnownLedger`
  - the `lastCheckpoint` corresponds to the last checkpoint ledger (reminder: a checkpoint ledger is one in which: `(ledger# + 1) mod 64 == 0`) and thus to a matching history archive upload.



## `start` State 
As you might expect, this state kicks off the FSM process.

There are a few possible branches in this state.

##### DB upgrade or fresh start
The "happiest path" is the following: either the ingestion database is empty, so we can start purely from scratch, or the state data in a database is outdated, meaning it needs to be upgraded and so can effectively be started from scratch after catch-up.

This branches differently depending on the last known ledger:

  - If it's newer than the last checkpoint, we need to wait for a new checkpoint to get the latest cumulative data. Note that though we probably *could* make incremental changes from block to block to the cumulative data, that would be more effort than it's worth relative to just waiting on the next history archive to get dumped. **Next state**: [`waitForCheckpoint`](#waitforcheckpoint-state).

  - If it's older, however, then we can just grok the missing gap (i.e. until the latest checkpoint) and build up (only) the time-series data. **Next state**: [`historyRange`](#historyrange-state).

In the other cases (matching last-known and last-checkpoint ledger, or no last-known), **next state**: [`build`](#build-state).

##### Otherwise
If we can't have a clean slate to work with, we need to fix partial state. Specifically,

  - If the last-known ledger is ahead of the last-ingested ledger, then Horizon is aware of much more data than it has processed. Here, we'll reset the time-series database and start over. **Next state**: [`start`](#start-state), with `lastIngestedLedger == 0`.

  - If the time-series database is newer than the last-known ledger (can occur if ingestion was done for a different range earlier, for example), then Horizon needs to become aware of the missing ledgers. **Next state**: [`historyRange`](#historyrange-state) from "last known" to "last stored" in time-series db.

**Next state**: [`resume`](#resume-state)


## `build` state
This is the big kahuna of the state machine: there aren't many state transitions aside from success/failure, and all roads ultimately should lead to ~~Rome~~ `build` in order to get ingestion done. This state only establishes a baseline for the cumulative data, though.


### Properties
This state tracks the:

  - `checkpointLedger`, which is Horizon's last-known (though possibly-unprocessed) checkpoint ledger, and 
  - `stop`, which optionally (though *universally*) transitions to the [`stop`](#stop-state) after this state is complete.

### Process
There's an important bit about this state: only one node should ever build, globally. Hence, there are a few checks at the start to ensure this.

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
  - 

**Next state**: [`start`](#start-state).

Otherwise, we have `ingestLedger == lastIngestedLedger + 1`, and will proceed to process the range from `ingestLedger` onward.

With the range prepared, only one other primary state transition is possible. If the last-known ledger of the Core backend is outdated relative to the above `ingestLedger`, we'll block until the requisite ledger is seen and processed by Core. **Next state**: [`resume`](#resume-state) again, with the last-processed ledger set to whatever is last-known to Core.

Otherwise, we can actually turn the ledger into time-series data: this is exactly the responsibility of `RunAllProcessorsOnLedger` and all of its subsequent friends. The deltas for the ledger(s) are ingested into the time-series db, then verified.

**Next state**: [`resume`](#resume-state) again, except now targeting the *next* ledger.


## `historyRange` state
The purpose of this state is to ingest a particular ledger range into the time-series database.

### Properties
This tracks an inclusive ledger range: [`fromLedger`, `toLedger`].


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
