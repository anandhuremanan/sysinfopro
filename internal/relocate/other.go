//go:build !windows

package relocate

import (
	"context"
	"errors"
)

func createJunction(link, target string) error { return errors.New("only available on Windows") }

func junctionTarget(path string) (string, bool) { return "", false }

func runningPrograms() map[string]bool { return map[string]bool{} }

func wslItems(ctx context.Context, records map[string]record, system string) []Item { return nil }

func moveWSL(ctx context.Context, name, base string) (*Result, error) {
	return nil, errors.New("only available on Windows")
}

func moveWSLBack(ctx context.Context, name string, rec record) (*Result, error) {
	return nil, errors.New("only available on Windows")
}
