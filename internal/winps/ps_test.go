//go:build integration

package winps

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRunMultiLineScript(t *testing.T) {
	// multi-line blocks and functions must run as one script, not line by line
	script := `
function Double($n) {
	$n * 2
}
[pscustomobject]@{
	value = Double 21
	list  = @(1, 2, 3)
} | ConvertTo-Json -Compress
`
	out, err := Run(context.Background(), script)
	if err != nil {
		t.Fatal(err)
	}

	got := struct {
		Value int   `json:"value"`
		List  []int `json:"list"`
	}{}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output %q is not the expected JSON: %v", out, err)
	}
	if got.Value != 42 || len(got.List) != 3 {
		t.Errorf("got %+v, want value 42 and three list items", got)
	}
}

func TestRunEmptyOutputIsAnError(t *testing.T) {
	if _, err := Run(context.Background(), "$null"); err == nil {
		t.Error("a query with no output should return an error, not empty data")
	}
}
