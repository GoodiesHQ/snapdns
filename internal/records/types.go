package records

import (
	"strings"

	"github.com/miekg/dns"
)

type RecordType string

const (
	RecordTypeA     RecordType = "a"
	RecordTypeAAAA  RecordType = "aaaa"
	RecordTypeCNAME RecordType = "cname"
	RecordTypeTXT   RecordType = "txt"
	RecordTypePTR   RecordType = "ptr"
	RecordTypeSRV   RecordType = "srv"
	RecordTypeMX    RecordType = "mx"
	RecordTypeNS    RecordType = "ns"
)

var dnsTypes = map[RecordType]uint16{
	RecordTypeA:     dns.TypeA,
	RecordTypeAAAA:  dns.TypeAAAA,
	RecordTypeCNAME: dns.TypeCNAME,
	RecordTypeTXT:   dns.TypeTXT,
	RecordTypePTR:   dns.TypePTR,
	RecordTypeSRV:   dns.TypeSRV,
	RecordTypeMX:    dns.TypeMX,
	RecordTypeNS:    dns.TypeNS,
}

func (rt RecordType) Normalize() RecordType {
	return RecordType(strings.ToLower(string(rt)))
}

func (rt RecordType) Type() uint16 {
	if value, found := dnsTypes[rt.Normalize()]; found {
		return value
	}
	return dns.TypeNone
}

func (rt RecordType) Valid() bool {
	return rt.Type() != dns.TypeNone
}
