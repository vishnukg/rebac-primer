// Unit tests for the in-memory TupleRepository adapter.
// Mirrors the Go suite in internal/authz/adapters/db/store_test.go.
//
// The store is a self-contained stateful unit with no collaborators, so no test
// doubles are needed: each test arranges real tuples, acts on the store, and
// asserts on its observable state.
import { describe, expect, it } from "vitest";
import makeInMemoryTupleRepository from "../src/authz-service/adapters/db/makeInMemoryTupleRepository.ts";
import type { TupleFilter } from "../src/authz-service/core/ports/tupleRepository.ts";
import { team, tuple } from "../src/shared/rebac.ts";
import { alice, bob, platformTeam, productWorkspace } from "./fixtures.ts";

const aliceMember = () => tuple(platformTeam, "member", alice);
const bobViewer = () => tuple(productWorkspace, "viewer", bob);

describe("makeInMemoryTupleRepository — has", () => {
    it("reports true for a seeded tuple", () => {
        // Arrange
        const t = aliceMember();
        const repository = makeInMemoryTupleRepository({ seed: [t] });

        // Act
        const found = repository.has(t.object, t.relation, t.user);

        // Assert
        expect(found).toBe(true);
    });

    it("reports false on an empty store", () => {
        // Arrange
        const repository = makeInMemoryTupleRepository();
        const t = aliceMember();

        // Act
        const found = repository.has(t.object, t.relation, t.user);

        // Assert
        expect(found).toBe(false);
    });

    it("reports true after the tuple is written", () => {
        // Arrange
        const repository = makeInMemoryTupleRepository();
        const t = aliceMember();

        // Act
        repository.write(t);

        // Assert
        expect(repository.has(t.object, t.relation, t.user)).toBe(true);
    });
});

describe("makeInMemoryTupleRepository — write / delete", () => {
    it("stores a duplicate write only once (idempotent)", () => {
        // Arrange
        const repository = makeInMemoryTupleRepository();
        const t = aliceMember();

        // Act
        repository.write(t);
        repository.write(t);

        // Assert
        expect(repository.findAll()).toHaveLength(1);
    });

    it("removes a stored tuple", () => {
        // Arrange
        const t = aliceMember();
        const repository = makeInMemoryTupleRepository({ seed: [t] });

        // Act
        repository.delete(t);

        // Assert
        expect(repository.has(t.object, t.relation, t.user)).toBe(false);
    });

    it("is a no-op when deleting a tuple that was never written", () => {
        // Arrange
        const repository = makeInMemoryTupleRepository({ seed: [aliceMember()] });

        // Act
        repository.delete(bobViewer());

        // Assert
        expect(repository.findAll()).toHaveLength(1);
    });
});

describe("makeInMemoryTupleRepository — queries", () => {
    it("findByObjectRelation returns only matching tuples", () => {
        // Arrange
        const match = bobViewer();
        const repository = makeInMemoryTupleRepository({ seed: [match, aliceMember()] });

        // Act
        const got = repository.findByObjectRelation(match.object, match.relation);

        // Assert
        expect(got).toEqual([match]);
    });

    const cases: { name: string; filter: TupleFilter | undefined; want: number }[] = [
        { name: "no filter matches all", filter: undefined, want: 2 },
        { name: "by object", filter: { object: platformTeam }, want: 1 },
        { name: "by relation", filter: { relation: "viewer" }, want: 1 },
        { name: "by object and relation", filter: { object: platformTeam, relation: "member" }, want: 1 },
        { name: "non-matching filter is empty", filter: { object: team("noSuchTeam") }, want: 0 },
    ];

    for (const c of cases) {
        it(`findAll ${c.name}`, () => {
            // Arrange
            const repository = makeInMemoryTupleRepository({ seed: [aliceMember(), bobViewer()] });

            // Act
            const got = repository.findAll(c.filter);

            // Assert
            expect(got).toHaveLength(c.want);
        });
    }
});
