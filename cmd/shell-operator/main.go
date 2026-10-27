package main

import (
	"fmt"
	"os"

	"github.com/deckhouse/deckhouse/pkg/log"
	"github.com/go-logr/logr/funcr"
	"github.com/spf13/cobra"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/flant/kube-client/klogtolog"
	"github.com/flant/shell-operator/pkg/app"
	"github.com/flant/shell-operator/pkg/debug"
	"github.com/flant/shell-operator/pkg/filter/jq"
)

func main() {
	// Build config from defaults and environment variables.
	// Flag defaults are set to these pre-merged values so an explicit CLI flag
	// wins, an env var wins over the hardcoded default, but no flag ever reads
	// the environment directly.
	cfg := app.NewConfig()
	if err := app.ParseEnv(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger()
	log.SetDefault(logger)

	// controller-runtime components (e.g. pkg/webhook/server's certwatcher)
	// use the generic logr interface, which is a separate sink from the
	// deckhouse logger above. Without this, controller-runtime buffers log
	// calls for ~30s then dumps a one-time diagnostic stack trace warning
	// that log.SetLogger was never called -- bridge it to our logger instead.
	ctrllog.SetLogger(funcr.New(func(prefix, args string) {
		if prefix != "" {
			logger.Info(prefix + ": " + args)
			return
		}
		logger.Info(args)
	}, funcr.Options{}))

	rootCmd := &cobra.Command{
		Use:   app.AppName,
		Short: fmt.Sprintf("%s %s: %s", app.AppName, app.Version, app.AppDescription),
		// klog adapter uses the final (post-parse) value of cfg.Debug.KubernetesAPI.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			klogtolog.InitAdapter(cfg.Debug.KubernetesAPI, logger.Named("klog"))
			return nil
		},
	}

	// version sub-command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version.",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("%s %s\n", app.AppName, app.Version)
			fl := jq.NewFilter()
			fmt.Println(fl.FilterInfo())
		},
	})

	// start sub-command (default)
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start shell-operator.",
		RunE:  start(logger, cfg),
	}
	applySliceFlags := app.BindFlags(cfg, rootCmd, startCmd)
	startCmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		applySliceFlags()
		// Sync the resolved debug socket path into the package-level CLI
		// default so any debug-* sub-command launched in the same process
		// (e.g. embedded test harnesses) sees the same value the operator
		// will bind to.
		debug.DefaultSocketPath = cfg.Debug.UnixSocket
		return nil
	}
	rootCmd.AddCommand(startCmd)

	debug.DefineDebugCommands(rootCmd)
	debug.DefineDebugCommandsSelf(rootCmd)

	// Make start the default command when no subcommand is given.
	rootCmd.RunE = start(logger, cfg)
	rootCmd.PreRunE = func(_ *cobra.Command, _ []string) error {
		applySliceFlags()
		debug.DefaultSocketPath = cfg.Debug.UnixSocket
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
