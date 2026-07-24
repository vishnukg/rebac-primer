package openfga_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"rebac-primer/internal/authz"
	"rebac-primer/internal/openfga"
	"rebac-primer/internal/rebac"
)

type writeRequestBody struct {
	AuthorizationModelID string `json:"authorization_model_id"`
	Writes               *struct {
		TupleKeys   []openfgaTuple `json:"tuple_keys"`
		OnDuplicate string         `json:"on_duplicate"`
	} `json:"writes"`
	Deletes *struct {
		TupleKeys []openfgaTuple `json:"tuple_keys"`
		OnMissing string         `json:"on_missing"`
	} `json:"deletes"`
}

type checkRequestBody struct {
	AuthorizationModelID string       `json:"authorization_model_id"`
	TupleKey             openfgaTuple `json:"tuple_key"`
}

type readRequestBody struct {
	TupleKey *openfgaTuple `json:"tuple_key"`
	Token    string        `json:"continuation_token"`
}

type openfgaTuple struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNew_GivenMissingConfig_WhenCalled_ThenReturnsError(t *testing.T) {
	// Arrange
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
			// Act
			_, err := openfga.New(tc.cfg)

			// Assert
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestListRelationships_GivenRelationOnlyFilter_WhenCalled_ThenReturnsError(t *testing.T) {
	// Arrange
	// The guard rejects the filter before any network call, so no client is needed.
	svc := &openfga.Service{}

	// Act
	_, err := svc.ListRelationships(t.Context(), authz.RelationshipFilter{
		Relation: rebac.RelationWorkspaceEditor,
	})

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "relation alone") {
		t.Errorf("error = %q, want it to mention filtering by relation alone", err)
	}
}

func TestWriteRelationships_GivenInvalidRelationship_WhenCalled_ThenReturnsValidationError(t *testing.T) {
	// Arrange
	svc := &openfga.Service{}

	// Act
	err := svc.WriteRelationships(t.Context(), []rebac.Relationship{{
		Resource: "roadmap",
		Relation: rebac.RelationDocumentOwner,
		Subject:  rebac.Subject(rebac.User("alice")),
	}})

	// Assert
	var validationErr *authz.RelationshipValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *authz.RelationshipValidationError, got %v", err)
	}
}

func TestDeleteRelationships_GivenInvalidRelationship_WhenCalled_ThenReturnsValidationError(t *testing.T) {
	// Arrange
	svc := &openfga.Service{}

	// Act
	err := svc.DeleteRelationships(t.Context(), []rebac.Relationship{{
		Resource: "roadmap",
		Relation: rebac.RelationDocumentOwner,
		Subject:  rebac.Subject(rebac.User("alice")),
	}})

	// Assert
	var validationErr *authz.RelationshipValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *authz.RelationshipValidationError, got %v", err)
	}
}

func TestWriteRelationships_UsesAtomicDuplicateIgnore(t *testing.T) {
	// Arrange
	const (
		storeID = "01H00000000000000000000000"
		modelID = "01H00000000000000000000001"
	)
	want := rebac.NewRelationship(
		rebac.Subject(rebac.User("alice")),
		rebac.RelationWorkspaceEditor,
		rebac.Workspace("productWorkspace"),
	)
	requests := 0
	var gotPath string
	var gotBody writeRequestBody
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    r,
		}, nil
	})}
	svc, err := openfga.New(openfga.Config{APIURL: "http://openfga.test", StoreID: storeID, ModelID: modelID, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Act
	if err := svc.WriteRelationships(t.Context(), []rebac.Relationship{want}); err != nil {
		t.Fatalf("WriteRelationships() returned unexpected error: %v", err)
	}

	// Assert
	if requests != 1 {
		t.Errorf("OpenFGA requests = %d, want 1 (no read-before-write)", requests)
	}
	if gotPath != "/stores/"+storeID+"/write" {
		t.Errorf("path = %q, want store write path", gotPath)
	}
	if gotBody.AuthorizationModelID != modelID {
		t.Errorf("authorization_model_id = %q, want %q", gotBody.AuthorizationModelID, modelID)
	}
	if gotBody.Writes == nil {
		t.Fatal("writes = nil, want a write payload")
	}
	if gotBody.Writes.OnDuplicate != "ignore" {
		t.Errorf("on_duplicate = %q, want ignore", gotBody.Writes.OnDuplicate)
	}
	wantTuple := openfgaTuple{
		User: string(want.Subject), Relation: string(want.Relation), Object: string(want.Resource),
	}
	if len(gotBody.Writes.TupleKeys) != 1 || gotBody.Writes.TupleKeys[0] != wantTuple {
		t.Errorf("tuple_keys = %+v, want [%+v]", gotBody.Writes.TupleKeys, wantTuple)
	}
}

func TestCheck_MapsRequestAndAllowedResponse(t *testing.T) {
	// Arrange
	const (
		storeID = "01H00000000000000000000000"
		modelID = "01H00000000000000000000001"
	)
	want := rebac.CheckRequest{
		Subject:    rebac.User("alice"),
		Permission: rebac.PermissionDocumentEdit,
		Resource:   rebac.Document("roadmap"),
	}
	var gotPath string
	var gotBody checkRequestBody
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"allowed":true}`)),
			Request:    r,
		}, nil
	})}
	svc, err := openfga.New(openfga.Config{APIURL: "http://openfga.test", StoreID: storeID, ModelID: modelID, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Act
	got, err := svc.Check(t.Context(), want)

	// Assert
	if err != nil {
		t.Fatalf("Check() returned unexpected error: %v", err)
	}
	if !got.Allowed {
		t.Error("Allowed = false, want true")
	}
	if gotPath != "/stores/"+storeID+"/check" {
		t.Errorf("path = %q, want store check path", gotPath)
	}
	if gotBody.AuthorizationModelID != modelID {
		t.Errorf("authorization_model_id = %q, want %q", gotBody.AuthorizationModelID, modelID)
	}
	wantTuple := openfgaTuple{
		User: string(want.Subject), Relation: string(want.Permission), Object: string(want.Resource),
	}
	if gotBody.TupleKey != wantTuple {
		t.Errorf("tuple_key = %+v, want %+v", gotBody.TupleKey, wantTuple)
	}
	if len(got.Trace) != 1 || !strings.Contains(got.Trace[0], "-> true") {
		t.Errorf("Trace = %v, want one allowed OpenFGA trace line", got.Trace)
	}
}

func TestListRelationships_FollowsPaginationAndMapsTuples(t *testing.T) {
	// Arrange
	const (
		storeID = "01H00000000000000000000000"
		modelID = "01H00000000000000000000001"
	)
	filter := authz.RelationshipFilter{
		Resource: rebac.Workspace("productWorkspace"),
		Relation: rebac.RelationWorkspaceEditor,
	}
	want := []rebac.Relationship{
		rebac.NewRelationship(rebac.Subject(rebac.User("alice")), filter.Relation, filter.Resource),
		rebac.NewRelationship(rebac.SubjectSet(rebac.Team("platform"), rebac.RelationTeamMember), filter.Relation, filter.Resource),
	}
	requests := 0
	var gotBodies []readRequestBody
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		var body readRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		gotBodies = append(gotBodies, body)

		responseBody := `{"tuples":[{"key":{"user":"user:alice","relation":"editor","object":"workspace:productWorkspace"},"timestamp":"2026-07-18T00:00:00Z"}],"continuation_token":"next-page"}`
		if requests == 2 {
			responseBody = `{"tuples":[{"key":{"user":"team:platform#member","relation":"editor","object":"workspace:productWorkspace"},"timestamp":"2026-07-18T00:00:00Z"}],"continuation_token":""}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Request:    r,
		}, nil
	})}
	svc, err := openfga.New(openfga.Config{APIURL: "http://openfga.test", StoreID: storeID, ModelID: modelID, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Act
	got, err := svc.ListRelationships(t.Context(), filter)

	// Assert
	if err != nil {
		t.Fatalf("ListRelationships() returned unexpected error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("OpenFGA requests = %d, want 2", requests)
	}
	if len(gotBodies) != 2 {
		t.Fatalf("captured request bodies = %d, want 2", len(gotBodies))
	}
	wantFilter := openfgaTuple{Relation: string(filter.Relation), Object: string(filter.Resource)}
	if gotBodies[0].TupleKey == nil || *gotBodies[0].TupleKey != wantFilter {
		t.Errorf("first tuple_key = %+v, want %+v", gotBodies[0].TupleKey, wantFilter)
	}
	if gotBodies[0].Token != "" || gotBodies[1].Token != "next-page" {
		t.Errorf("continuation tokens = [%q %q], want [empty next-page]", gotBodies[0].Token, gotBodies[1].Token)
	}
	if len(got) != len(want) {
		t.Fatalf("tuples = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tuple[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDeleteRelationships_UsesAtomicMissingIgnore(t *testing.T) {
	// Arrange
	const (
		storeID = "01H00000000000000000000000"
		modelID = "01H00000000000000000000001"
	)
	want := rebac.NewRelationship(
		rebac.Subject(rebac.User("bob")),
		rebac.RelationWorkspaceViewer,
		rebac.Workspace("productWorkspace"),
	)
	var gotBody writeRequestBody
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    r,
		}, nil
	})}
	svc, err := openfga.New(openfga.Config{APIURL: "http://openfga.test", StoreID: storeID, ModelID: modelID, HTTPClient: httpClient})
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Act
	if err := svc.DeleteRelationships(t.Context(), []rebac.Relationship{want}); err != nil {
		t.Fatalf("DeleteRelationships() returned unexpected error: %v", err)
	}

	// Assert
	if gotBody.Deletes == nil {
		t.Fatal("deletes = nil, want a delete payload")
	}
	if gotBody.Deletes.OnMissing != "ignore" {
		t.Errorf("on_missing = %q, want ignore", gotBody.Deletes.OnMissing)
	}
	wantTuple := openfgaTuple{
		User: string(want.Subject), Relation: string(want.Relation), Object: string(want.Resource),
	}
	if len(gotBody.Deletes.TupleKeys) != 1 || gotBody.Deletes.TupleKeys[0] != wantTuple {
		t.Errorf("tuple_keys = %+v, want [%+v]", gotBody.Deletes.TupleKeys, wantTuple)
	}
}
