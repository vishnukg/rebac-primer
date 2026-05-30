package shared_test

import (
	"testing"

	"rebac-primer/internal/shared"
)

// These tests cover the ReBAC vocabulary primitives in the shared package:
// object/subject construction, parsing, and the subject-set predicate.
//
// The units under test are pure functions with no collaborators, so there are
// no test doubles here — stubs and mocks only earn their keep when a unit talks
// to a port (see internal/authz/authz_test.go for that distinction).

func TestParseObject_GivenWellFormedReference_WhenParsed_ThenReturnsTypeAndID(t *testing.T) {
	// Arrange
	const input = "workspace:productWorkspace"

	// Act
	typ, id, err := shared.ParseObject(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseObject(%q) returned unexpected error: %v", input, err)
	}
	if typ != shared.ObjectTypeWorkspace {
		t.Errorf("type = %q, want %q", typ, shared.ObjectTypeWorkspace)
	}
	if id != "productWorkspace" {
		t.Errorf("id = %q, want %q", id, "productWorkspace")
	}
}

func TestParseObject_GivenIDContainingColon_WhenParsed_ThenSplitsOnFirstColonOnly(t *testing.T) {
	// Arrange: only the first colon separates type from id.
	const input = "document:a:b:c"

	// Act
	typ, id, err := shared.ParseObject(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseObject(%q) returned unexpected error: %v", input, err)
	}
	if typ != shared.ObjectTypeDocument || id != "a:b:c" {
		t.Errorf("got (type=%q, id=%q), want (document, a:b:c)", typ, id)
	}
}

func TestParseObject_GivenMalformedReference_WhenParsed_ThenReturnsError(t *testing.T) {
	// Arrange
	cases := map[string]string{
		"empty string":      "",
		"no separator":      "user",
		"empty type":        ":alice",
		"empty id":          "user:",
		"unrecognised type": "robot:r2d2",
		"separator only":    ":",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, _, err := shared.ParseObject(input)

			// Assert
			if err == nil {
				t.Errorf("ParseObject(%q) = nil error, want an error", input)
			}
		})
	}
}

func TestSubjectSet_GivenObjectAndRelation_WhenBuilt_ThenFormatsAsObjectHashRelation(t *testing.T) {
	// Arrange
	obj := shared.Team("platformTeam")

	// Act
	got := shared.SubjectSet(obj, shared.RelationTeamMember)

	// Assert
	if want := shared.Subject("team:platformTeam#member"); got != want {
		t.Errorf("SubjectSet() = %q, want %q", got, want)
	}
}

func TestParseSubjectSet_GivenSubjectSet_WhenParsed_ThenSplitsObjectAndRelation(t *testing.T) {
	// Arrange
	input := shared.SubjectSet(shared.Team("platformTeam"), shared.RelationTeamMember)

	// Act
	obj, rel, err := shared.ParseSubjectSet(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseSubjectSet(%q) returned unexpected error: %v", input, err)
	}
	if obj != shared.Team("platformTeam") {
		t.Errorf("object = %q, want %q", obj, shared.Team("platformTeam"))
	}
	if rel != shared.RelationTeamMember {
		t.Errorf("relation = %q, want %q", rel, shared.RelationTeamMember)
	}
}

func TestParseSubjectSet_GivenMalformedSubjectSet_WhenParsed_ThenReturnsError(t *testing.T) {
	// Arrange
	cases := map[string]shared.Subject{
		"no hash":        "team:platformTeam",
		"empty object":   "#member",
		"empty relation": "team:platformTeam#",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, _, err := shared.ParseSubjectSet(input)

			// Assert
			if err == nil {
				t.Errorf("ParseSubjectSet(%q) = nil error, want an error", input)
			}
		})
	}
}

func TestIsSubjectSet_GivenSubjectSet_WhenChecked_ThenReportsTrue(t *testing.T) {
	// Arrange
	subject := shared.SubjectSet(shared.Team("platformTeam"), shared.RelationTeamMember)

	// Act
	got := shared.IsSubjectSet(subject)

	// Assert
	if !got {
		t.Errorf("IsSubjectSet(%q) = false, want true", subject)
	}
}

func TestIsSubjectSet_GivenPlainObject_WhenChecked_ThenReportsFalse(t *testing.T) {
	// Arrange
	subject := shared.Subject(shared.User("alice"))

	// Act
	got := shared.IsSubjectSet(subject)

	// Assert
	if got {
		t.Errorf("IsSubjectSet(%q) = true, want false", subject)
	}
}

func TestObjectConstructor_GivenEmptyID_WhenBuilt_ThenPanics(t *testing.T) {
	// Arrange: the constructors guard against empty ids, which would produce an
	// ambiguous "user:" reference.
	defer func() {
		// Assert
		if r := recover(); r == nil {
			t.Error("User(\"\") = no panic, want a panic on empty id")
		}
	}()

	// Act
	_ = shared.User("")
}

func TestTuple_GivenParts_WhenBuilt_ThenPopulatesAllFields(t *testing.T) {
	// Arrange
	object := shared.Workspace("productWorkspace")
	subject := shared.SubjectSet(shared.Team("platformTeam"), shared.RelationTeamMember)

	// Act
	got := shared.Tuple(object, shared.RelationWorkspaceEditor, subject)

	// Assert
	want := shared.TupleKey{
		Object:   object,
		Relation: shared.RelationWorkspaceEditor,
		User:     subject,
	}
	if got != want {
		t.Errorf("Tuple() = %+v, want %+v", got, want)
	}
}
