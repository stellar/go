package ledgerbackend

import (
	"context"
	"fmt"

	"github.com/stellar/go/support/log"
)

type catchupStream struct {
	dir            workingDir
	from           uint32
	to             uint32
	coreCmdFactory coreCmdFactory
	log            *log.Entry
	useDB          bool
}

func newCatchupStream(r *stellarCoreRunner, from, to uint32) catchupStream {
	// We want to use ephemeral directories in running the catchup command
	// (used for the reingestion use case) because it's possible to run parallel
	// reingestion where multiple captive cores are active on the same machine.
	// Having ephemeral directories  will ensure that each ingestion worker will
	// have a separate working directory
	dir := newWorkingDir(r, true)
	return catchupStream{
		dir:            dir,
		from:           from,
		to:             to,
		coreCmdFactory: newCoreCmdFactory(r, dir),
		log:            r.log,
		useDB:          r.useDB,
	}
}

func (s catchupStream) getWorkingDir() workingDir {
	return s.dir
}

func (s catchupStream) start(ctx context.Context) (cmdI, pipe, error) {
	var err error
	var cmd cmdI
	var captiveCorePipe pipe

	rangeArg := fmt.Sprintf("%d/%d", s.to, s.to-s.from+1)
	params := []string{"catchup", rangeArg, "--metadata-output-stream", s.coreCmdFactory.getPipeName()}

	// horizon operator has specified to use external storage for captive core ledger state
	// instruct captive core invocation to not use memory, and in that case
	// cc will look at DATABASE property in cfg toml for the external storage source to use.
	// when using external storage of ledgers, use new-db to first set the state of
	// remote db storage to genesis to purge any prior state and reset.
	if s.useDB {
		cmd, err = s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOffline, true, "new-db")
		if err != nil {
			return nil, pipe{}, fmt.Errorf("error creating command: %w", err)
		}
		if err = cmd.Run(); err != nil {
			return nil, pipe{}, fmt.Errorf("error initializing core db: %w", err)
		}
	} else {
		params = append(params, "--in-memory")
	}

	cmd, err = s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOffline, true, params...)
	if err != nil {
		return nil, pipe{}, fmt.Errorf("error creating command: %w", err)
	}

	captiveCorePipe, err = s.coreCmdFactory.startCaptiveCore(cmd)
	if err != nil {
		return nil, pipe{}, fmt.Errorf("error starting `stellar-core run` subprocess: %w", err)
	}

	return cmd, captiveCorePipe, nil
}
