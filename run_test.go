package layerengine

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDeepRecursionIsCatchable verifies the stack bound turns runaway
// recursion into a returned error rather than a crash.
func TestDeepRecursionIsCatchable(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function recurse(n)
  if n <= 0 then return 0 end
  return 1 + recurse(n - 1)
end
`)
	if err != nil {
		t.Fatalf("compile recurse: %v", err)
	}

	layer := &Layer{
		Name:    "recurse",
		Input:   []string{"n"},
		Output:  []LayerOutput{{Name: "out", Type: "number"}},
		FnProto: proto,
	}

	res, err := runLayer(layer, []any{1_000_000})
	if err == nil {
		t.Fatalf("expected an overflow error, got result %v", res)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "overflow") {
		t.Fatalf("expected an overflow error, got: %v", err)
	}
}

// TestGuardMemoryCancelsOnBreach verifies the memory watchdog cancels the
// layer context (with the right cause) once the heap grows past the limit.
func TestGuardMemoryCancelsOnBreach(t *testing.T) {
	cause := errors.New("memory breach")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	go guardMemory(ctx, cancel, 8<<20, cause) // 8 MiB

	// Keep allocating and holding 1 MiB chunks until the watchdog trips.
	// Growing past the limit, rather than allocating once, keeps the test
	// robust to the baseline sample racing with the allocation.
	var sink [][]byte
	deadline := time.After(5 * time.Second)
loop:
	for {
		chunk := make([]byte, 1<<20)
		for i := 0; i < len(chunk); i += 4096 {
			chunk[i] = 1 // touch pages so the allocation is real
		}
		sink = append(sink, chunk)

		select {
		case <-ctx.Done():
			break loop
		case <-deadline:
			t.Fatal("guardMemory did not cancel within 5s")
		default:
		}
		time.Sleep(60 * time.Millisecond) // give the 50ms poller a chance
	}

	runtime.KeepAlive(sink)
	if got := context.Cause(ctx); got != cause {
		t.Fatalf("wrong cancel cause: got %v, want %v", got, cause)
	}
}
