package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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

	flag.Parse()

	// for file comparison, we don't need to execute any functionality
	if aJsonFilename != "" && bJsonFilename != "" {
		aJsonRaw, err := os.ReadFile(aJsonFilename)
		if err != nil {
			log.Fatalf("unable to read file %q", aJsonFilename)
		}

		bJsonRaw, err := os.ReadFile(bJsonFilename)
		if err != nil {
			log.Fatalf("unable to read file %q", bJsonFilename)
		}

		var aSnap, bSnap lookups.Snapshot

		if err := json.Unmarshal(aJsonRaw, &aSnap); err != nil {
			log.Fatalf("invalid json snapshot from %q", aJsonFilename)
		}
		if err := json.Unmarshal(bJsonRaw, &bSnap); err != nil {
			log.Fatalf("invalid json snapshot from %q", bJsonFilename)
		}

		lookups.Compare(&aSnap, &bSnap)
		return
	}

	if configFilename == "" {
		log.Fatal("configuration is required")
	}
	if group == "" {
		log.Fatal("resolver group is required")
	}

	fmt.Println("Filename: " + configFilename)

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	var resolver *config.Resolver
	for i := range cfg.Resolvers {
		if cfg.Resolvers[i].Name == group {
			resolver = &cfg.Resolvers[i]
			break
		}
	}
	if resolver == nil {
		log.Fatalf("resolver group %q does not exist in %s", group, configFilename)
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

		for _, recordType := range recordTypes {
			res, err := lookups.Query(
				context.Background(),
				resolver.Servers,
				fqdn,
				recordType,
			)
			if err != nil {
				log.Fatalf("dns query failed: %v", err)
			}

			snapshot.Results = append(snapshot.Results, res)
		}
	}

	snapshotJson, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}

	if outFilename == "" {
		fmt.Println(string(snapshotJson))
	} else {
		os.WriteFile(outFilename, snapshotJson, 0644)
	}
}
