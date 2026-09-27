package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

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
	if aJsonFilename != "" && bJsonFilename != "" {
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

	var resolver *config.Resolver
	for i := range cfg.Resolvers {
		if cfg.Resolvers[i].Name == group {
			resolver = &cfg.Resolvers[i]
			break
		}
	}
	if resolver == nil {
		slog.Error("resolver group does not exist", "config", configFilename, "group", group)
		os.Exit(1)
	}

	var snapshot = lookups.Snapshot{
		Domain:        cfg.Domain,
		ResolverGroup: group,
		Timestamp:     time.Now(),
		Results:       make([]lookups.Result, 0),
	}

	for name, recordTypes := range cfg.Records {
		var fqdn string
		if name == "@" {
			fqdn = strings.Trim(cfg.Domain, ".")
		} else {
			fqdn = strings.TrimSuffix(name, ".") + "." + strings.TrimLeft(cfg.Domain, ".")
		}
		slog.Debug("looking up records", "fqdn", fqdn, "types", recordTypes)

		for _, recordType := range recordTypes {
			res, err := lookups.Query(
				context.Background(),
				resolver.Servers,
				fqdn,
				recordType,
			)
			if err != nil {
				slog.Error("dns query failed", "err", err)
				os.Exit(1)
			}

			snapshot.Results = append(snapshot.Results, res)
		}
	}

	snapshotJson, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		slog.Error("failed to marshal", "err", err)
		os.Exit(1)
	}

	if outFilename == "" {
		fmt.Println(string(snapshotJson))
	} else {
		os.WriteFile(outFilename, snapshotJson, 0644)
		slog.Debug("completed snapshot", "output", outFilename)
	}
}
