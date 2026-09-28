package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/goodieshq/snapdns/internal/config"
	"github.com/goodieshq/snapdns/internal/lookups"
)

func main() {
	const configDefault = "config.yml"
	var configFilename string
	var outFilename string
	var group string
	var aJsonFilename, bJsonFilename string
	var verbose bool

	// yaml configuration
	flag.StringVar(&configFilename, "config", configDefault, "Configuration YAML file")
	flag.StringVar(&configFilename, "c", configDefault, "alias for --config")
	// dns server group to use for resolution (e.g. provider nameservers)
	flag.StringVar(&group, "group", "", "dns resolver group to use (defined in configuration)")
	flag.StringVar(&group, "g", "", "alias for --group")
	// output filename
	flag.StringVar(&outFilename, "output", "", "output json filename")
	flag.StringVar(&outFilename, "o", "", "alias for --output")
	// json comparison
	flag.StringVar(&aJsonFilename, "a", "", "used for comparing two json output files (requires -b)")
	flag.StringVar(&bJsonFilename, "b", "", "used for comparing two json output files (requires -a)")
	// verbosity
	flag.BoolVar(&verbose, "verbose", false, "Enable debug output")
	flag.BoolVar(&verbose, "v", false, "alias for --output")

	flag.Parse()

	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	// for file comparison, we don't need to execute any functionality
	if aJsonFilename != "" || bJsonFilename != "" {
		if aJsonFilename == "" || bJsonFilename == "" {
			slog.Error("both -a and -b must be provided to perform a comparison")
			os.Exit(1)
		}

		aJsonRaw, err := os.ReadFile(aJsonFilename)
		if err != nil {
			slog.Error("unable to read file", "filename", aJsonFilename)
			os.Exit(1)
		}

		bJsonRaw, err := os.ReadFile(bJsonFilename)
		if err != nil {
			slog.Error("unable to read file", "filename", bJsonFilename)
			os.Exit(1)
		}

		var aSnap, bSnap lookups.Snapshot

		if err := json.Unmarshal(aJsonRaw, &aSnap); err != nil {
			slog.Error("invalid json snapshot", "filename", aJsonFilename)
			os.Exit(1)
		}
		if err := json.Unmarshal(bJsonRaw, &bSnap); err != nil {
			slog.Error("invalid json snapshot", "filename", bJsonFilename)
			os.Exit(1)
		}

		lookups.Compare(&aSnap, &bSnap)
		return
	}

	if configFilename == "" {
		slog.Error("configuration is required")
		os.Exit(1)
	}
	if group == "" {
		slog.Error("resolver group is required")
		os.Exit(1)
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		slog.Error("failed to read the configuration", "config", configFilename, "err", err)
		os.Exit(1)
	}
	slog.Debug("loaded snapdns configuration", "config", configFilename)

	ctx := context.Background()
	snapshot, err := lookups.GetSnapshot(ctx, cfg, group)
	if err != nil {
		slog.Error("failed to get snapshot", "err", err)
		os.Exit(1)
	}

	snapshotJson, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		slog.Error("failed to marshal snapshot to JSON", "err", err)
		os.Exit(1)
	}

	if outFilename == "" {
		fmt.Println(string(snapshotJson))
	} else {
		os.WriteFile(outFilename, snapshotJson, 0644)
		slog.Debug("completed snapshot", "output", outFilename)
	}
}
