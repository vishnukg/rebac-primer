package graph_test

import (
	"errors"
	"testing"

	"rebac-primer/internal/authzservice/adapters/graph"
)

func TestResult_OK_IsOKReturnsTrue(t *testing.T) {
	r := graph.OK(42)
	if !r.IsOK() {
		t.Error("expected IsOK()=true for OK result")
	}
	if r.Err() != nil {
		t.Errorf("expected Err()=nil for OK result, got %v", r.Err())
	}
}

func TestResult_Fail_IsOKReturnsFalse(t *testing.T) {
	r := graph.Fail[int](errors.New("something broke"))
	if r.IsOK() {
		t.Error("expected IsOK()=false for Fail result")
	}
	if r.Err() == nil {
		t.Error("expected Err() != nil for Fail result")
	}
}

func TestResult_Value_ReturnsTrueForOK(t *testing.T) {
	r := graph.OK("hello")
	v, ok := r.Value()
	if !ok {
		t.Error("expected ok=true")
	}
	if v != "hello" {
		t.Errorf("expected value %q, got %q", "hello", v)
	}
}

func TestResult_Value_ReturnsFalseForFail(t *testing.T) {
	r := graph.Fail[string](errors.New("oops"))
	v, ok := r.Value()
	if ok {
		t.Error("expected ok=false for Fail result")
	}
	if v != "" {
		t.Errorf("expected zero value for failed result, got %q", v)
	}
}

func TestResult_Unwrap_ReturnsValueForOK(t *testing.T) {
	r := graph.OK(99)
	got := r.Unwrap()
	if got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestResult_Unwrap_PanicsForFail(t *testing.T) {
	r := graph.Fail[int](errors.New("broken"))
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected Unwrap to panic on a Fail result")
		}
	}()
	r.Unwrap()
}

func TestMap_TransformsValueOnSuccess(t *testing.T) {
	r := graph.OK(3)
	mapped := graph.Map(r, func(n int) string {
		return "x" + string(rune('0'+n))
	})
	if !mapped.IsOK() {
		t.Error("expected Map on OK result to return OK")
	}
	v, _ := mapped.Value()
	if v != "x3" {
		t.Errorf("expected %q, got %q", "x3", v)
	}
}

func TestMap_PropagatesFailureWithoutCallingF(t *testing.T) {
	called := false
	r := graph.Fail[int](errors.New("upstream error"))
	mapped := graph.Map(r, func(n int) string {
		called = true
		return "should not run"
	})
	if called {
		t.Error("expected f not to be called on a Fail result")
	}
	if mapped.IsOK() {
		t.Error("expected Map on Fail result to remain a failure")
	}
}

func TestCollect_AllOK_ReturnsSlice(t *testing.T) {
	results := []graph.Result[int]{graph.OK(1), graph.OK(2), graph.OK(3)}
	combined := graph.Collect(results)
	if !combined.IsOK() {
		t.Fatalf("expected Collect to succeed, got err: %v", combined.Err())
	}
	vals, _ := combined.Value()
	if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
		t.Errorf("unexpected values: %v", vals)
	}
}

func TestCollect_FirstFailureShortCircuits(t *testing.T) {
	results := []graph.Result[int]{
		graph.OK(1),
		graph.Fail[int](errors.New("second failed")),
		graph.OK(3),
	}
	combined := graph.Collect(results)
	if combined.IsOK() {
		t.Error("expected Collect to fail when any result is a failure")
	}
	if combined.Err().Error() != "second failed" {
		t.Errorf("expected error %q, got %q", "second failed", combined.Err().Error())
	}
}
