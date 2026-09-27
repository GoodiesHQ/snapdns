package lookups

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/goodieshq/snapdns/internal/config"
	"github.com/goodieshq/snapdns/internal/records"
	"github.com/miekg/dns"
	"golang.org/x/sync/errgroup"
)

type Snapshot struct {
	Domain        string    `json:"domain"`
	ResolverGroup string    `json:"resolver_group"`
	Timestamp     time.Time `json:"timestamp"`
	Results       []Result  `json:"results"`
}

func getResolver(cfg *config.Config, resolverGroup string) *config.Resolver {
	for i := range cfg.Resolvers {
		if cfg.Resolvers[i].Name == resolverGroup {
			return &cfg.Resolvers[i]
		}
	}
	return nil
}

func buildFqdn(name, base string) string {
	var fqdn string
	if name == "@" {
		fqdn = strings.Trim(base, ".")
	} else {
		fqdn = strings.TrimSuffix(name, ".") + "." + strings.TrimLeft(base, ".")
	}
	return dns.Fqdn(fqdn)
}

func GetSnapshot(ctx context.Context, cfg *config.Config, resolverGroup string) (*Snapshot, error) {
	resolver := getResolver(cfg, resolverGroup)
	if resolver == nil {
		return nil, fmt.Errorf("resolver group does not exist: %s", resolverGroup)
	}

	type job struct {
		fqdn       string
		recordType records.RecordType
	}

	var jobs []job
	for name, recordTypes := range cfg.Records {
		fqdn := buildFqdn(name, cfg.Domain)
		for _, recordType := range recordTypes {
			jobs = append(jobs, job{fqdn, recordType})
		}
	}

	results := make([]Result, len(jobs))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(16)

	servers := resolver.Servers

	for i, job := range jobs {
		g.Go(func() error {
			slog.Debug("performing lookup", "fqdn", job.fqdn, "type", job.recordType)
			res, err := Query(ctx, servers, job.fqdn, job.recordType)
			if err != nil {
				return err
			}
			results[i] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &Snapshot{
		Domain:        cfg.Domain,
		ResolverGroup: resolverGroup,
		Timestamp:     time.Now(),
		Results:       results,
	}, nil
}
