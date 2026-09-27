package config

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goodieshq/snapdns/internal/records"
)

type Config struct {
	Domain    string                          `yaml:"domain"` // base domain for all DNS lookups
	Resolvers []Resolver                      `yaml:"resolvers"`
	Records   map[string][]records.RecordType `yaml:"records"` //
}

type Resolver struct {
	Name    string   `yaml:"name"`
	Servers []string `yaml:"servers"`
}

func (cfg *Config) hasRecords() bool {
	for record := range cfg.Records {
		if len(cfg.Records[record]) > 0 {
			return true
		}
	}
	return false
}

func ReadConfig(filename string) (*Config, error) {
	var cfg Config

	// read the configuration file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// parse the yaml config into the config struct
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// normalize and trim
	cfg.Domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(cfg.Domain)), ".")
	if cfg.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	for resolverIndex := range cfg.Resolvers {
		resolver := &cfg.Resolvers[resolverIndex]

		for serverIndex, server := range resolver.Servers {
			normalizedServer, err := normalizeServer(server)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid DNS server in resolver group %q: %w",
					resolver.Name,
					err,
				)
			}

			resolver.Servers[serverIndex] = normalizedServer
		}
	}

	for name, recordTypes := range cfg.Records {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("record name cannot be empty")
		}

		normalized := normalize(recordTypes)
		if len(normalized) == 0 {
			return nil, fmt.Errorf("no record types configured for %q", name)
		}

		seen := make(map[records.RecordType]struct{}, len(normalized))
		for _, recordType := range normalized {
			if !recordType.Valid() {
				return nil, fmt.Errorf("invalid record type %q for %q", recordType, name)
			}
			if _, exists := seen[recordType]; exists {
				return nil, fmt.Errorf("duplicate record type %q for %q", recordType, name)
			}
			seen[recordType] = struct{}{}

			if recordType == records.RecordTypeCNAME &&
				(slices.Contains(normalized, records.RecordTypeA) || slices.Contains(normalized, records.RecordTypeAAAA)) {
				return nil, fmt.Errorf("cname cannot be combined with a/aaaa records: %q", name)
			}
		}

		cfg.Records[name] = normalized
	}

	// ensure at least one valid record is provided
	if !cfg.hasRecords() {
		return nil, fmt.Errorf("at least one record is required")
	}

	return &cfg, nil
}

func normalize(recordTypes []records.RecordType) []records.RecordType {
	recordTypesNormalized := make([]records.RecordType, len(recordTypes))
	for i := range recordTypes {
		recordTypesNormalized[i] = records.RecordType(strings.ToLower(string(recordTypes[i])))
	}
	return recordTypesNormalized
}

func normalizeServer(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", fmt.Errorf("DNS server cannot be empty")
	}

	// Explicit port: hostname:port, IPv4:port, or [IPv6]:port.
	host, port, err := net.SplitHostPort(server)
	if err == nil {
		if host == "" {
			return "", fmt.Errorf("DNS server %q has no host", server)
		}

		portNumber, err := strconv.ParseUint(port, 10, 16)
		if err != nil || portNumber == 0 {
			return "", fmt.Errorf("DNS server %q has an invalid port", server)
		}

		return net.JoinHostPort(host, port), nil
	}

	if strings.HasPrefix(server, "[") || strings.HasSuffix(server, "]") {
		return "", fmt.Errorf(
			"DNS server %q is invalid; bracketed IPv6 addresses require a port",
			server,
		)
	}

	return net.JoinHostPort(server, "53"), nil
}
