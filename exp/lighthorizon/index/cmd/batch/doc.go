// Package batch provides two commands: map and reduce that can be run in AWS
// Batch to generate indexes for occurences of accounts in each checkpoint.
//
// map step is using AWS_BATCH_JOB_ARRAY_INDEX env variable provided by AWS
// Batch to cut all checkpoint history into smaller chunks, each processed by a
// single map batch job (and by multiple parallel workers in a single job). A
// single job simply creates indexes for a given range of checkpoints and save
// indexes and all accounts seen in a given range (FlushAccounts method) to a
// job folder (job_X, X = 0, 1, 2, 3, ...) in S3.
//
//	               network history split into chunks:
//	[  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  ]
//	                  ----
//	                 /    \
//	                /      \
//	               /        \
//	              [..........] <- each chunk consists of checkpoints
//	                   |
//	                   . - each checkpoint is processed by a free
//	                       worker (go routine)
//
// reduce step is responsible for merging all indexes created in map step into a
// final indexes for each account and for entire network history. Each reduce
// job goes through all map job results (0..MAP_JOBS) and reads all accounts
// processed in a given map job. Then for each account it merges indexes from
// all map jobs. Each reduce job maintains `doneAccounts` map because if a given
// account index was processed earlier it should be skipped instead of being
// processed again. Each reduce job also runs multiple parallel workers. Finally
// the method that is used to determine if the following (job, worker) should
// process a given account is using a 64-bit hash of account ID. The hash is
// split into two 32-bit parts: left and right. If the left part modulo
// REDUCE_JOBS is equal the job index and the right part modulo a number of
// parallel workers is equal the worker index then the account is processed.
// Otherwise it's skipped (and will be picked by another (job, worker) pair).
//
//	            map step results saved in S3:
//	x x x x x x x x x x x x x x x x x x x x x x x x x x x x
//	|
//	ㄴ job0/accounts  <- each job results contains a list of accounts
//	|                   processed by a given job...
//	|
//	ㄴ job0/...       <- ...and partial indexes
//
//	hash(account_id) => XXXX YYYY <-  64 bit hash of account id is calculated
//
//	if XXXX % REDUCE_JOBS == JOB_ID and YYYY % WORKERS_COUNT = WORKER_ID
//	then process a given account by merging all indexes of a given account
//	in all map step results, then mark account as done so if the account
//	is seen again it will be skiped,
//
//	else: skip the account.
package batch
