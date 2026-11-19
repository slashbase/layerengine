package layerengine

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/slashbase/layerengine/modules"
	lua "github.com/yuin/gopher-lua"
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

	res, err := runLayer(context.Background(), layer, []any{1_000_000}, []modules.Module{})
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

// TestCyclicTableOutputIsAnErrorNotACrash covers the generated-code path: a
// layer can build a self-referencing table, and returning it has to fail the
// layer rather than overflow the stack and take the process down.
func TestCyclicTableOutputIsAnErrorNotACrash(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function loop()
  local t = {}
  t.self = t
  return t
end
`)
	if err != nil {
		t.Fatalf("compile loop: %v", err)
	}

	layer := &Layer{
		Name:    "loop",
		Output:  []LayerOutput{{Name: "out", Type: "table"}},
		FnProto: proto,
	}

	_, err = runLayer(context.Background(), layer, nil, []modules.Module{})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("runLayer error = %v, want a cycle error", err)
	}
}

// TestRunLayerContextCancelsRunningChunk proves the caller's context reaches the
// Lua state: a chunk that loops forever has to stop on cancel rather than wait
// out the 60s per-layer timeout.
func TestRunLayerContextCancelsRunningChunk(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function spin()
  while true do end
end
`)
	if err != nil {
		t.Fatalf("compile spin: %v", err)
	}

	layer := &Layer{Name: "spin", FnProto: proto}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := runLayer(ctx, layer, nil, []modules.Module{})
		done <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runLayer error = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runLayer did not stop within 5s of cancellation")
	}
}

// tableRecorder is a module that records every table Init builds, so a test can
// see how many layer states the engine asks it for.
type tableRecorder struct {
	mu     sync.Mutex
	tables []*lua.LTable
}

func (*tableRecorder) Name() string { return "recorder" }

func (*tableRecorder) Usage() string {
	return "marker:string = recorder.marker() - a fixed marker"
}

func (*tableRecorder) Manifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Functions: []modules.Function{{
			Name:    "marker",
			Doc:     "a fixed marker",
			Returns: []modules.Return{{Name: "marker", Type: "string"}},
		}},
	}
}

func (r *tableRecorder) Init(l *lua.LState) *lua.LTable {
	table := l.NewTable()
	table.RawSetString("marker", l.NewFunction(func(l *lua.LState) int {
		l.Push(lua.LString("ok"))
		return 1
	}))

	r.mu.Lock()
	r.tables = append(r.tables, table)
	r.mu.Unlock()
	return table
}

// TestEachLayerStateGetsItsOwnModuleTable pins the contract Module states: Init
// runs once per layer state, so state kept on the returned table is per-layer
// while the module value itself is shared.
func TestEachLayerStateGetsItsOwnModuleTable(t *testing.T) {
	recorder := &tableRecorder{}
	engine := NewBlankLayerEngine()
	if err := engine.LoadModules([]modules.Module{recorder}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	if err := engine.LoadLayers([]Layer{{
		Name:   "marked",
		Output: []LayerOutput{{Name: "marker", Type: "string"}},
		Code:   "function marked() return recorder.marker() end",
	}}); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}

	for range 2 {
		output, err := engine.RunLayer("marked", nil)
		if err != nil {
			t.Fatalf("RunLayer: %v", err)
		}
		values, ok := output.([]any)
		if !ok || len(values) != 1 || values[0] != "ok" {
			t.Fatalf("RunLayer output = %#v, want one \"ok\"", output)
		}
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.tables) < 2 {
		t.Fatalf("Init was called %d times for two runs, want one table per layer state", len(recorder.tables))
	}
	seen := make(map[*lua.LTable]struct{}, len(recorder.tables))
	for _, table := range recorder.tables {
		if _, duplicate := seen[table]; duplicate {
			t.Fatal("Init returned the same table for two layer states")
		}
		seen[table] = struct{}{}
	}
}
