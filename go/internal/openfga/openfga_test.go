package openfga_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/openfga"
	"rebac-primer/internal/rebac"
)

func TestNew_GivenMissingConfig_WhenCalled_ThenReturnsError(t *testing.T) {
	cases := map[string]struct {
		cfg  openfga.Config
		want string
	}{
		"api url":  {openfga.Config{StoreID: "store", ModelID: "model"}, "APIURL"},
		"store id": {openfga.Config{APIURL: "http://127.0.0.1:8080", ModelID: "model"}, "StoreID"},
		"model id": {openfga.Config{APIURL: "http://127.0.0.1:8080", StoreID: "store"}, "ModelID"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := openfga.New(tc.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestWriteTuples_GivenInvalidTuple_WhenCalled_ThenReturnsValidationError(t *testing.T) {
	svc := &openfga.Service{}
	err := svc.WriteTuples(context.Background(), []rebac.TupleKey{{
		Object:   "roadmap",
		Relation: rebac.RelationDocumentOwner,
		User:     rebac.Subject(rebac.User("alice")),
	}})

	var validationErr *authz.TupleValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *authz.TupleValidationError, got %v", err)
	}
}
