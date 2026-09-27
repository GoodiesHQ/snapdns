package lookups

import (
	"context"
	"fmt"
	"time"

	"github.com/goodieshq/snapdns/internal/records"
	"github.com/miekg/dns"
)

type Snapshot struct {
	Domain        string    `json:"domain"`
	ResolverGroup string    `json:"resolver_group"`
	Timestamp     time.Time `json:"timestamp"`
	Results       []Result  `json:"results"`
}

type Result struct {
	Name    string             `json:"name"`
	Type    records.RecordType `json:"type"`
	Server  string             `json:"server"`
	RCode   string             `json:"rcode"`
	Answers []string           `json:"answers"`
}

func Query(ctx context.Context, servers []string, fqdn string, rt records.RecordType) (Result, error) {
	if len(servers) == 0 {
		return Result{}, fmt.Errorf("no DNS servers configured")
	}

	fqdn = dns.Fqdn(fqdn)

	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, rt.Type())
	msg.RecursionDesired = true

	udpClient := &dns.Client{
		Net:     "udp",
		Timeout: 3 * time.Second,
	}
	tcpClient := &dns.Client{
		Net:     "tcp",
		Timeout: 3 * time.Second,
	}

	var errLast error

	for _, server := range servers {
		response, _, err := udpClient.ExchangeContext(ctx, msg, server)
		if err != nil {
			errLast = err
			continue
		}

		// use tcp client if response is truncated
		if response.Truncated {
			response, _, err = tcpClient.ExchangeContext(ctx, msg, server)
			if err != nil {
				errLast = err
				continue
			}
		}

		answers := make([]string, 0, len(response.Answer))
		for _, ans := range response.Answer {
			answers = append(answers, ans.String())
		}

		return Result{
			Name:    fqdn,
			Type:    rt,
			Server:  server,
			RCode:   dns.RcodeToString[response.Rcode],
			Answers: answers,
		}, nil
	}

	return Result{}, fmt.Errorf("all configured DNS servers failed (last error %w)", errLast)
}
