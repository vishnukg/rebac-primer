package authz_test

import (
	"errors"
	"testing"

	"rebac-primer/internal/authz"
)

func TestResult_OK_IsOKReturnsTrue(t *testing.T) {
	// Arrange
	r := authz.OK(42)

	// Act + Assert
	if !r.IsOK() {
		t.Error("expected IsOK()=true for OK result")
	}
	if r.Err() != nil {
		t.Errorf("expected Err()=nil for OK result, got %v", r.Err())
	}
}

func TestResult_Fail_IsOKReturnsFalse(t *testing.T) {
	// Arrange
	r := authz.Fail[int](errors.New("something broke"))

	// Act + Assert
	if r.IsOK() {
		t.Error("expected IsOK()=false for Fail result")
	}
	if r.Err() == nil {
		t.Error("expected Err() != nil for Fail result")
	}
}

func TestResult_Value_ReturnsTrueForOK(t *testing.T) {
	// Arrange
	r := authz.OK("hello")

	// Act
	v, ok := r.Value()

	// Assert
	if !ok {
		t.Error("expected ok=true")
	}
	if v != "hello" {
		t.Errorf("expected value %q, got %q", "hello", v)
	}
}

func TestResult_Value_ReturnsFalseForFail(t *testing.T) {
	// Arrange
	r := authz.Fail[string](errors.New("oops"))

	// Act
	v, ok := r.Value()

	// Assert
	if ok {
		t.Error("expected ok=false for Fail result")
	}
	if v != "" {
		t.Errorf("expected zero value for failed result, got %q", v)
	}
}

func TestResult_Unwrap_ReturnsValueForOK(t *testing.T) {
	// Arrange
	r := authz.OK(99)

	// Act + Assert: no panic expected
	got := r.Unwrap()
	if got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

func TestResult_Unwrap_PanicsForFail(t *testing.T) {
	// Arrange
	r := authz.Fail[int](errors.New("broken"))

	// Act + Assert: Unwrap on a failure must panic.
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected Unwrap to panic on a Fail result")
		}
	}()
	r.Unwrap()
}

func TestMap_TransformsValueOnSuccess(t *testing.T) {
	// Arrange
	r := authz.OK(3)

	// Act: double the value inside the Result.
	mapped := authz.Map(r, func(n int) string {
		return "x" + string(rune('0'+n)) // "x3"
	})

	// Assert
	if !mapped.IsOK() {
		t.Error("expected Map on OK result to return OK")
	}
	v, _ := mapped.Value()
	if v != "x3" {
		t.Errorf("expected %q, got %q", "x3", v)
	}
}

func TestMap_PropagatesFailureWithoutCallingF(t *testing.T) {
	// Arrange
	called := false
	r := authz.Fail[int](errors.New("upstream error"))

	// Act
	mapped := authz.Map(r, func(n int) string {
		called = true
		return "should not run"
	})

	// Assert
	if called {
		t.Error("expected f not to be called on a Fail result")
	}
	if mapped.IsOK() {
		t.Error("expected Map on Fail result to remain a failure")
	}
}

func TestCollect_AllOK_ReturnsSlice(t *testing.T) {
	// Arrange
	results := []authz.Result[int]{
		authz.OK(1),
		authz.OK(2),
		authz.OK(3),
	}

	// Act
	combined := authz.Collect(results)

	// Assert
	if !combined.IsOK() {
		t.Fatalf("expected Collect to succeed, got err: %v", combined.Err())
	}
	vals, _ := combined.Value()
	if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
		t.Errorf("unexpected values: %v", vals)
	}
}

func TestCollect_FirstFailureShortCircuits(t *testing.T) {
	// Arrange
	results := []authz.Result[int]{
		authz.OK(1),
		authz.Fail[int](errors.New("second failed")),
		authz.OK(3),
	}

	// Act
	combined := authz.Collect(results)

	// Assert
	if combined.IsOK() {
		t.Error("expected Collect to fail when any result is a failure")
	}
	if combined.Err().Error() != "second failed" {
		t.Errorf("expected error %q, got %q", "second failed", combined.Err().Error())
	}
}
