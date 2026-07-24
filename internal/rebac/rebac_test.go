package rebac_test

import (
	"testing"

	"rebac-primer/internal/rebac"
)

// These tests cover the ReBAC vocabulary primitives in the shared package:
// resource/subject construction, parsing, and the subject-set predicate.
//
// The units under test are pure functions with no collaborators, so there are
// no test doubles here — stubs and mocks only earn their keep when a unit talks
// to a port (see internal/authz/authz_test.go for that distinction).

func TestParseResource_GivenWellFormedReference_WhenParsed_ThenReturnsTypeAndID(t *testing.T) {
	// Arrange
	const input = "workspace:productWorkspace"

	// Act
	typ, id, err := rebac.ParseResource(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseResource(%q) returned unexpected error: %v", input, err)
	}
	if typ != rebac.ResourceTypeWorkspace {
		t.Errorf("type = %q, want %q", typ, rebac.ResourceTypeWorkspace)
	}
	if id != "productWorkspace" {
		t.Errorf("id = %q, want %q", id, "productWorkspace")
	}
}

func TestParseResource_GivenIDContainingColon_WhenParsed_ThenSplitsOnFirstColonOnly(t *testing.T) {
	// Arrange: only the first colon separates type from id.
	const input = "document:a:b:c"

	// Act
	typ, id, err := rebac.ParseResource(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseResource(%q) returned unexpected error: %v", input, err)
	}
	if typ != rebac.ResourceTypeDocument || id != "a:b:c" {
		t.Errorf("got (type=%q, id=%q), want (document, a:b:c)", typ, id)
	}
}

func TestParseResource_GivenMalformedReference_WhenParsed_ThenReturnsError(t *testing.T) {
	// Arrange
	cases := map[string]string{
		"empty string":      "",
		"no separator":      "user",
		"empty type":        ":alice",
		"empty id":          "user:",
		"blank id":          "workspace: ",
		"unrecognised type": "robot:r2d2",
		"separator only":    ":",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, _, err := rebac.ParseResource(input)

			// Assert
			if err == nil {
				t.Errorf("ParseResource(%q) = nil error, want an error", input)
			}
		})
	}
}

func TestSubjectSet_GivenObjectAndRelation_WhenBuilt_ThenFormatsAsObjectHashRelation(t *testing.T) {
	// Arrange
	resource := rebac.Team("platformTeam")

	// Act
	got := rebac.SubjectSet(resource, rebac.RelationTeamMember)

	// Assert
	if want := rebac.Subject("team:platformTeam#member"); got != want {
		t.Errorf("SubjectSet() = %q, want %q", got, want)
	}
}

func TestParseSubjectSet_GivenSubjectSet_WhenParsed_ThenSplitsObjectAndRelation(t *testing.T) {
	// Arrange
	input := rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember)

	// Act
	resource, rel, err := rebac.ParseSubjectSet(input)

	// Assert
	if err != nil {
		t.Fatalf("ParseSubjectSet(%q) returned unexpected error: %v", input, err)
	}
	if resource != rebac.Team("platformTeam") {
		t.Errorf("resource = %q, want %q", resource, rebac.Team("platformTeam"))
	}
	if rel != rebac.RelationTeamMember {
		t.Errorf("relation = %q, want %q", rel, rebac.RelationTeamMember)
	}
}

func TestParseSubjectSet_GivenMalformedSubjectSet_WhenParsed_ThenReturnsError(t *testing.T) {
	// Arrange
	cases := map[string]rebac.Subject{
		"no hash":        "team:platformTeam",
		"empty resource": "#member",
		"empty relation": "team:platformTeam#",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			_, _, err := rebac.ParseSubjectSet(input)

			// Assert
			if err == nil {
				t.Errorf("ParseSubjectSet(%q) = nil error, want an error", input)
			}
		})
	}
}

func TestIsSubjectSet_GivenSubjectSet_WhenChecked_ThenReportsTrue(t *testing.T) {
	// Arrange
	subject := rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember)

	// Act
	got := rebac.IsSubjectSet(subject)

	// Assert
	if !got {
		t.Errorf("IsSubjectSet(%q) = false, want true", subject)
	}
}

func TestIsSubjectSet_GivenPlainObject_WhenChecked_ThenReportsFalse(t *testing.T) {
	// Arrange
	subject := rebac.Subject(rebac.User("alice"))

	// Act
	got := rebac.IsSubjectSet(subject)

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
	_ = rebac.User("")
}

func TestNewRelationship_GivenParts_WhenBuilt_ThenPopulatesAllFields(t *testing.T) {
	// Arrange
	resource := rebac.Workspace("productWorkspace")
	subject := rebac.SubjectSet(rebac.Team("platformTeam"), rebac.RelationTeamMember)

	// Act
	got := rebac.NewRelationship(subject, rebac.RelationWorkspaceEditor, resource)

	// Assert
	want := rebac.Relationship{
		Subject:  subject,
		Relation: rebac.RelationWorkspaceEditor,
		Resource: resource,
	}
	if got != want {
		t.Errorf("NewRelationship() = %+v, want %+v", got, want)
	}
}

// FuzzParseResource exercises the parser in the package that owns it.
// Run with: go test -fuzz=FuzzParseResource -fuzztime=30s ./internal/rebac
func FuzzParseResource(f *testing.F) {
	f.Add("user:alice")
	f.Add("team:platformTeam")
	f.Add("workspace:productWorkspace")
	f.Add("document:roadmapDocument")
	f.Add("")
	f.Add(":")
	f.Add("user:")
	f.Add("workspace: ")
	f.Add(":alice")
	f.Add("unknown:something")

	f.Fuzz(func(t *testing.T, s string) {
		typ, id, err := rebac.ParseResource(s)
		if err != nil {
			return
		}
		var resource rebac.Resource
		switch typ {
		case rebac.ResourceTypeUser:
			resource = rebac.User(id)
		case rebac.ResourceTypeTeam:
			resource = rebac.Team(id)
		case rebac.ResourceTypeWorkspace:
			resource = rebac.Workspace(id)
		case rebac.ResourceTypeDocument:
			resource = rebac.Document(id)
		default:
			t.Fatalf("ParseResource returned unrecognised type %q", typ)
		}
		if string(resource) != s {
			t.Errorf("round-trip failed: ParseResource(%q) -> type=%s id=%s -> Resource=%q", s, typ, id, resource)
		}
	})
}
