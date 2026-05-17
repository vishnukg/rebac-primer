package authz

import (
	"context"
	"errors"
	"testing"

	openfga "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
)

type fakeOpenFGAClient struct {
	checkReq    fgaclient.ClientCheckRequest
	checkResp   *fgaclient.ClientCheckResponse
	checkErr    error
	wroteTuples []fgaclient.ClientTupleKey
	writeErr    error
}

func (f *fakeOpenFGAClient) Check(_ context.Context, req fgaclient.ClientCheckRequest) (*fgaclient.ClientCheckResponse, error) {
	f.checkReq = req
	if f.checkErr != nil {
		return nil, f.checkErr
	}
	return f.checkResp, nil
}

func (f *fakeOpenFGAClient) WriteTuples(_ context.Context, tuples []fgaclient.ClientTupleKey) error {
	f.wroteTuples = tuples
	return f.writeErr
}

func TestOpenFGAAuthorizer_CheckMapsRequestAndResult(t *testing.T) {
	// Arrange
	resp := &fgaclient.ClientCheckResponse{}
	resp.SetAllowed(true)
	fake := &fakeOpenFGAClient{checkResp: resp}
	auth := newOpenFGAAuthorizerWithClient(fake)

	// Act
	result, err := auth.Check(context.Background(), CheckRequest{
		User:     User("alice"),
		Relation: RelationDocumentCanRead,
		Object:   Document("roadmapDocument"),
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatal("expected OpenFGA allowed response to map to allowed=true")
	}
	if len(result.Trace) != 1 || result.Trace[0] != "OpenFGA evaluated the relationship graph remotely" {
		t.Fatalf("unexpected trace: %#v", result.Trace)
	}
	if fake.checkReq.User != "user:alice" {
		t.Errorf("got user %q", fake.checkReq.User)
	}
	if fake.checkReq.Relation != "can_read" {
		t.Errorf("got relation %q", fake.checkReq.Relation)
	}
	if fake.checkReq.Object != "document:roadmapDocument" {
		t.Errorf("got object %q", fake.checkReq.Object)
	}
}

func TestOpenFGAAuthorizer_CheckWrapsSDKError(t *testing.T) {
	// Arrange
	fake := &fakeOpenFGAClient{checkErr: errors.New("network unavailable")}
	auth := newOpenFGAAuthorizerWithClient(fake)

	// Act
	_, err := auth.Check(context.Background(), CheckRequest{
		User:     User("alice"),
		Relation: RelationDocumentCanEdit,
		Object:   Document("roadmapDocument"),
	})

	// Assert
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fake.checkErr) {
		t.Fatalf("expected wrapped SDK error, got %v", err)
	}
}

func TestOpenFGAAuthorizer_WriteTuplesMapsRepoTuplesToSDKTuples(t *testing.T) {
	// Arrange
	fake := &fakeOpenFGAClient{}
	auth := newOpenFGAAuthorizerWithClient(fake)
	tuples := []TupleKey{
		Tuple(Document("roadmapDocument"), RelationDocumentOwner, Subject(User("alice"))),
	}

	// Act
	err := auth.WriteTuples(context.Background(), tuples)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.wroteTuples) != 1 {
		t.Fatalf("expected one tuple write, got %d", len(fake.wroteTuples))
	}
	want := *openfga.NewTupleKey("user:alice", "owner", "document:roadmapDocument")
	if fake.wroteTuples[0] != want {
		t.Fatalf("got tuple %#v, want %#v", fake.wroteTuples[0], want)
	}
}

func TestOpenFGAAuthorizer_WriteTuplesWrapsSDKError(t *testing.T) {
	// Arrange
	fake := &fakeOpenFGAClient{writeErr: errors.New("write failed")}
	auth := newOpenFGAAuthorizerWithClient(fake)

	// Act
	err := auth.WriteTuples(context.Background(), []TupleKey{
		Tuple(Document("roadmapDocument"), RelationDocumentOwner, Subject(User("alice"))),
	})

	// Assert
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, fake.writeErr) {
		t.Fatalf("expected wrapped SDK error, got %v", err)
	}
}
