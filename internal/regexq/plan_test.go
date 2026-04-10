package regexq

import (
	"regexp"
	"testing"
)

func TestPlanLiteralPrefix(t *testing.T) {
	re := regexp.MustCompile("MAX_FILE_SIZE.*foo")
	p := Plan("MAX_FILE_SIZE.*foo", re)
	if len(p.Terms) == 0 {
		t.Fatalf("expected terms")
	}
	if len(p.Pairs) == 0 {
		t.Fatalf("expected pairs")
	}
}

func TestPlanNoPrefix(t *testing.T) {
	re := regexp.MustCompile(".*foo")
	p := Plan(".*foo", re)
	if len(p.Terms) != 0 {
		t.Fatalf("expected no terms for non-literal prefix")
	}
}
