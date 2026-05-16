# Practice: Collab Docs Lite

Build a small API for collaborative documents.

Requirements:

- users can belong to teams
- teams can have workspace viewer/editor access
- documents inherit workspace access
- document owners can delete documents
- editors can update content
- viewers can read and comment

Tasks:

1. Add an HTTP layer with Node's built-in `http` module or a small framework.
2. Protect each handler with `Authorizer.check`.
3. Add tuple writes when a document is created.
4. Add tests for read, edit, delete, and denied access.
5. Add one integration test against a real OpenFGA server.

The goal is to design the graph first, then let the application code follow it.
