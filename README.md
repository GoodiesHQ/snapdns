# snapdns

A lightweight utility for capturing and comparing DNS record snapshots across multiple resolvers.

## Overview

snapdns allows you to:
- **Snapshot mode**: Capture current DNS records for a domain from configured resolvers and records
- **Compare mode**: Compare two snapshots to identify changes in DNS records

The intention for this was originally to identify any misconfigrations from performing a domain/nameserver migration, but it could be used for any DNS record comparison use case.

## Installation

    go build -o snapdns .

## Configuration

It is required to provide: a base domain (all records will be used relative to this base), at least one group of nameservers to use as resolvers, and individual records and their record types.

For example:

    domain: example.com
    resolvers:
        - name: public
          servers: ["1.1.1.1", "8.8.8.8"]
    records:
        "@": [a, aaaa, txt, mx]
        www: [a, aaaa]
        _dmarc: [txt]

## Usage:

### Snapshot Mode
You must provide a configuration YAML file (`-c`/`--config`) and a resolver group name (`-g`/`--group`) (e.g. "public" from the above configuration). By default, it will output to the terminal, but you can provide an output file (`-o`/`--out`) and it will write in JSON format.

    ./snapdns -config config.yml -out snapshot.json -g public

### Compare Mode
You must provide two JSON snapshot files (`-a` and `-b`). The output will show any differences in the records between the two snapshots or any records that exist in one and not the other. It only reports on differences between records, not between metadata such as DNS servers.

    ./snapdns -a old.json -b new.json
