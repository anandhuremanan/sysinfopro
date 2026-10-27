package winexec

import (
	"context"
	"testing"
	"time"
)

func TestDefaultDeadline(t *testing.T) {
	cmd, cancel := Command(context.Background(), "go", "version")
	defer cancel()
	if cmd.Cancel == nil {
		t.Fatal("command is not tied to a context")
	}
	ctx, c2 := context.WithTimeout(context.Background(), time.Second)
	defer c2()
	_, cancel3 := Command(ctx, "go", "version")
	cancel3()
}

func TestOutput(t *testing.T) {
	out, err := Output(context.Background(), "go", "version")
	if err != nil || len(out) == 0 {
		t.Fatalf("go version: %v %q", err, out)
	}
}
