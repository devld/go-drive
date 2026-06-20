package script

import "testing"

func TestParsePoolConfigRejectsInvalidLimits(t *testing.T) {
	for _, value := range []string{
		"0,0,0,30m",
		"1,-1,0,30m",
		"1,1,-1,30m",
		"1,1,2,30m",
	} {
		t.Run(value, func(t *testing.T) {
			if _, e := parsePoolConfig(value); e == nil {
				t.Fatal("expected invalid pool config to fail")
			}
		})
	}
}
