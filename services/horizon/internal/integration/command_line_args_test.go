package integration

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	horizon "github.com/stellar/go/services/horizon/internal"
	"github.com/stretchr/testify/assert"
	"io"
	stdLog "log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestIngestionFilteringAlwaysDefaultingToTrue(t *testing.T) {
	t.Run("ingestion filtering flag set to default value", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{})
		err := test.StartHorizon()
		assert.NoError(t, err)
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
	})
	t.Run("ingestion filtering flag set to false", func(t *testing.T) {
		test := NewParameterTest(t, map[string]string{"exp-enable-ingestion-filtering": "false"})
		err := test.StartHorizon()
		assert.NoError(t, err)
		assert.Equal(t, test.HorizonIngest().Config().EnableIngestionFiltering, true)
	})
}

func TestDeprecatedOutputForIngestionFilteringFlag(t *testing.T) {
	storeStdout := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	stdLog.SetOutput(os.Stdout)

	test := NewParameterTest(t, map[string]string{"exp-enable-ingestion-filtering": "false"})
	if innerErr := test.StartHorizon(); innerErr != nil {
		t.Fatalf("Failed to start Horizon: %v", innerErr)
	}

	// Use a wait group to wait for the goroutine to finish before proceeding
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.Close(); err != nil {
			t.Errorf("Failed to close Stdout")
			return
		}
	}()

	// Give some time for the goroutine to start
	time.Sleep(time.Millisecond)

	outputBytes, _ := io.ReadAll(r)
	wg.Wait() // Wait for the goroutine to finish before proceeding

	os.Stdout = storeStdout

	assert.Contains(t, string(outputBytes), "DEPRECATED - No ingestion filter rules are defined by default, which equates to "+
		"no filtering of historical data. If you have never added filter rules to this deployment, then nothing further needed. "+
		"If you have defined ingestion filter rules prior but disabled filtering overall by setting this flag disabled with "+
		"--exp-enable-ingestion-filtering=false, then you should now delete the filter rules using the Horizon Admin API to achieve "+
		"the same no-filtering result. Remove usage of this flag in all cases.")
}

func TestHelpOutputForNoIngestionFilteringFlag(t *testing.T) {
	config, flags := horizon.Flags()

	horizonCmd := &cobra.Command{
		Use:           "horizon",
		Short:         "Client-facing api server for the Stellar network",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long:          "Client-facing API server for the Stellar network.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := horizon.NewAppFromFlags(config, flags)
			if err != nil {
				return err
			}
			return nil
		},
	}

	var writer io.Writer = &bytes.Buffer{}
	horizonCmd.SetOutput(writer)

	horizonCmd.SetArgs([]string{"-h"})
	if err := flags.Init(horizonCmd); err != nil {
		fmt.Println(err)
	}
	if err := horizonCmd.Execute(); err != nil {
		fmt.Println(err)
	}

	output := writer.(*bytes.Buffer).String()
	assert.NotContains(t, output, "--exp-enable-ingestion-filtering")
}
