package ledgerbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/log"
)

type runFromStream struct {
	dir                     workingDir
	from                    uint32
	hash                    string
	coreCmdFactory          coreCmdFactory
	log                     *log.Entry
	useDB                   bool
	captiveCoreNewDBCounter prometheus.Counter
}

func newRunFromStream(r *stellarCoreRunner, from uint32, hash string, captiveCoreNewDBCounter prometheus.Counter) runFromStream {
	// We only use ephemeral directories on windows because there is
	// no way to terminate captive core gracefully on windows.
	// Having an ephemeral directory ensures that it is wiped out
	// whenever we terminate captive core
	dir := newWorkingDir(r, runtime.GOOS == "windows")
	return runFromStream{
		dir:                     dir,
		from:                    from,
		hash:                    hash,
		coreCmdFactory:          newCoreCmdFactory(r, dir),
		log:                     r.log,
		useDB:                   r.useDB,
		captiveCoreNewDBCounter: captiveCoreNewDBCounter,
	}
}

func (s runFromStream) getWorkingDir() workingDir {
	return s.dir
}

func (s runFromStream) offlineInfo(ctx context.Context) (stellarcore.InfoResponse, error) {
	cmd, err := s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOnline, false, "offline-info")
	if err != nil {
		return stellarcore.InfoResponse{}, fmt.Errorf("error creating offline-info cmd: %w", err)
	}
	output, err := cmd.Output()
	if err != nil {
		return stellarcore.InfoResponse{}, fmt.Errorf("error executing offline-info cmd: %w", err)
	}
	var info stellarcore.InfoResponse
	err = json.Unmarshal(output, &info)
	if err != nil {
		return stellarcore.InfoResponse{}, fmt.Errorf("invalid output of offline-info cmd: %w", err)
	}
	return info, nil
}

func (s runFromStream) start(ctx context.Context) (cmd cmdI, captiveCorePipe pipe, returnErr error) {
	var err error
	var createNewDB bool
	defer func() {
		if returnErr != nil && createNewDB {
			// if we could not start captive core remove the new db we created
			s.dir.remove()
		}
	}()
	if s.useDB {
		// Check if on-disk core DB exists and what's the LCL there. If not what
		// we need remove storage dir and start from scratch.
		var info stellarcore.InfoResponse
		info, err = s.offlineInfo(ctx)
		if err != nil {
			s.log.Infof("Error running offline-info: %v, removing existing storage-dir contents", err)
			createNewDB = true
		} else if info.Info.Ledger.Num <= 1 || uint32(info.Info.Ledger.Num) > s.from {
			s.log.Infof("Unexpected LCL in Stellar-Core DB: %d (want: %d), removing existing storage-dir contents", info.Info.Ledger.Num, s.from)
			createNewDB = true
		}

		if createNewDB {
			if s.captiveCoreNewDBCounter != nil {
				s.captiveCoreNewDBCounter.Inc()
			}
			if err = s.dir.remove(); err != nil {
				return nil, pipe{}, fmt.Errorf("error removing existing storage-dir contents: %w", err)
			}

			cmd, err = s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOnline, true, "new-db")
			if err != nil {
				return nil, pipe{}, fmt.Errorf("error creating command: %w", err)
			}

			if err = cmd.Run(); err != nil {
				return nil, pipe{}, fmt.Errorf("error initializing core db: %w", err)
			}

			// Do a quick catch-up to set the LCL in core to be our expected starting
			// point.
			if s.from > 2 {
				cmd, err = s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOnline, true, "catchup", fmt.Sprintf("%d/0", s.from-1))
			} else {
				cmd, err = s.coreCmdFactory.newCmd(ctx, stellarCoreRunnerModeOnline, true, "catchup", "2/0")
			}
			if err != nil {
				return nil, pipe{}, fmt.Errorf("error creating command: %w", err)
			}

			if err = cmd.Run(); err != nil {
				return nil, pipe{}, fmt.Errorf("error runing stellar-core catchup: %w", err)
			}
		}

		cmd, err = s.coreCmdFactory.newCmd(
			ctx,
			stellarCoreRunnerModeOnline,
			true,
			"run",
			"--metadata-output-stream", s.coreCmdFactory.getPipeName(),
		)
	} else {
		cmd, err = s.coreCmdFactory.newCmd(
			ctx,
			stellarCoreRunnerModeOnline,
			true,
			"run",
			"--in-memory",
			"--start-at-ledger", fmt.Sprintf("%d", s.from),
			"--start-at-hash", s.hash,
			"--metadata-output-stream", s.coreCmdFactory.getPipeName(),
		)
	}
	if err != nil {
		return nil, pipe{}, fmt.Errorf("error creating command: %w", err)
	}

	captiveCorePipe, err = s.coreCmdFactory.startCaptiveCore(cmd)
	if err != nil {
		return nil, pipe{}, fmt.Errorf("error starting `stellar-core run` subprocess: %w", err)
	}

	return cmd, captiveCorePipe, nil
}
