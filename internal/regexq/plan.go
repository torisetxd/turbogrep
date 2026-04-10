package regexq

import "regexp"

type QueryPlan struct {
	Terms [][3]byte
	Pairs [][2][3]byte
}

func Plan(pattern string, re *regexp.Regexp) QueryPlan {
	if re == nil {
		re = regexp.MustCompile(pattern)
	}
	prefix, _ := re.LiteralPrefix()
	if len(prefix) < 3 {
		return QueryPlan{}
	}

	terms := make([][3]byte, 0, len(prefix)-2)
	for i := 0; i+2 < len(prefix); i++ {
		terms = append(terms, [3]byte{prefix[i], prefix[i+1], prefix[i+2]})
	}

	pairs := make([][2][3]byte, 0, len(terms)-1)
	for i := 0; i+1 < len(terms); i++ {
		pairs = append(pairs, [2][3]byte{terms[i], terms[i+1]})
	}

	return QueryPlan{Terms: dedupe(terms), Pairs: pairs}
}

func dedupe(in [][3]byte) [][3]byte {
	seen := make(map[[3]byte]struct{}, len(in))
	out := make([][3]byte, 0, len(in))
	for _, t := range in {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
