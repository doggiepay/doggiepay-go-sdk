package doggiepay

import "strings"

// NormalizeChain maps common chain aliases to DoggiePay Integration aliases.
func NormalizeChain(alias string) string {
	a := strings.TrimSpace(alias)
	switch strings.ToLower(a) {
	case "ethereum", "eth":
		return "ETH"
	case "polygon", "matic", "pol":
		return "MATIC"
	case "base":
		return "BASE"
	case "arbitrum", "arb":
		return "ARB"
	case "avalanche", "avax":
		return "AVAX"
	case "optimism", "op":
		return "OP"
	default:
		return a
	}
}

// NormalizeChains maps and de-duplicates a slice of chain aliases.
func NormalizeChains(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, c := range in {
		n := NormalizeChain(c)
		if n == "" {
			continue
		}
		k := strings.ToUpper(n)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, n)
	}
	return out
}
