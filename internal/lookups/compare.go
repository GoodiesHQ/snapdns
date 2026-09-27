package lookups

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

func Compare(a, b *Snapshot) {
	aResults := indexResults(a.Results)
	bResults := indexResults(b.Results)

	// get a list of all keys in either set of results
	keys := make(map[string]struct{}, len(aResults)+len(bResults))
	for key := range aResults {
		keys[key] = struct{}{}
	}
	for key := range bResults {
		keys[key] = struct{}{}
	}

	sortedKeys := make([]string, 0, len(keys))
	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	differencesFound := false

	// iterate over sorted keys in both results
	for _, key := range sortedKeys {
		aResult, aExists := aResults[key]
		bResult, bExists := bResults[key]

		switch {
		case !aExists:
			differencesFound = true
			fmt.Printf("Only in snapshot B: %s %s\n", bResult.Name, bResult.Type)

		case !bExists:
			differencesFound = true
			fmt.Printf("Only in snapshot A: %s %s\n", aResult.Name, aResult.Type)

		default:
			if aResult.RCode != bResult.RCode {
				differencesFound = true
				fmt.Printf(
					"RCode changed for %s %s: A=%s, B=%s\n",
					aResult.Name,
					aResult.Type,
					aResult.RCode,
					bResult.RCode,
				)
			}

			aAnswers := answerSet(aResult.Answers)
			bAnswers := answerSet(bResult.Answers)

			onlyA := difference(aAnswers, bAnswers)
			onlyB := difference(bAnswers, aAnswers)

			if len(onlyA) == 0 && len(onlyB) == 0 {
				continue
			}

			differencesFound = true
			fmt.Printf("Answer difference for %s %s:\n", aResult.Name, aResult.Type)

			for _, answer := range onlyA {
				fmt.Printf("  [A] %s\n", answer)
			}
			for _, answer := range onlyB {
				fmt.Printf("  [B] %s\n", answer)
			}
		}
	}

	if !differencesFound {
		fmt.Println("No DNS record differences found.")
	}
}

func answerSet(answers []string) map[string]struct{} {
	set := make(map[string]struct{}, len(answers))

	for _, answer := range answers {
		set[normalizeAnswer(answer)] = struct{}{}
	}

	return set
}

func normalizeAnswer(answer string) string {
	rr, err := dns.NewRR(answer)
	if err != nil {
		// preserve malformed/unexpected answers so they still compare
		return strings.TrimSpace(answer)
	}

	// TTL is not relevant for a snapshot comparison
	rr.Header().Ttl = 0

	return rr.String()
}

func indexResults(results []Result) map[string]Result {
	indexed := make(map[string]Result, len(results))

	for _, result := range results {
		key := strings.ToLower(dns.Fqdn(result.Name)) + "\x00" + string(result.Type)
		indexed[key] = result
	}

	return indexed
}

func difference(a, b map[string]struct{}) []string {
	values := make([]string, 0)

	for value := range a {
		if _, exists := b[value]; !exists {
			values = append(values, value)
		}
	}

	sort.Strings(values)
	return values
}
