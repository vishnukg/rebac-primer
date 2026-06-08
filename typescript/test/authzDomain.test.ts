// Unit tests for the AuthZ domain service (makeAuthzService) in isolation.
// Mirrors the Go suite in internal/authz/authz_test.go.
//
// The domain is a thin orchestrator over two driven ports — TupleRepository and
// Evaluator — which makes it the right place to demonstrate the difference
// between stubs and mocks:
//
//   - A STUB stands in for a collaborator and returns canned answers. It is used
//     for STATE verification: "given the evaluator says allowed, does check()
//     return allowed?" The test never inspects how the stub was called.
//
//   - A MOCK also stands in for a collaborator but records the calls it received.
//     It is used for BEHAVIOUR verification: "does writeTuples() call
//     repository.write once per tuple, with the exact tuples, in order?" The
//     assertions are about the interaction, not a returned value.
//
// Both kinds implement the same port interface; the difference is what the test
// asserts on, not the type.
import { describe, expect, it } from "vitest";
import makeAuthzService from "../src/authz-service/core/domain/makeAuthzService.ts";
import type { TupleFilter, TupleRepository } from "../src/authz-service/core/ports/tupleRepository.ts";
import type { Evaluator } from "../src/authz-service/core/ports/evaluator.ts";
import type { CheckRequest, CheckResult, TupleKey } from "../src/shared/rebac.ts";
import { document, tuple, workspace } from "../src/shared/rebac.ts";
import { alice } from "./fixtures.ts";

// ── Stubs (state verification) ──────────────────────────────────────────────

// A STUB evaluator returning a fixed result, recording nothing.
const stubEvaluator = (result: CheckResult): Evaluator => ({
    evaluate: async () => result,
});

// A STUB evaluator that always fails.
const failingEvaluator = (error: Error): Evaluator => ({
    evaluate: async () => {
        throw error;
    },
});

// A STUB repository whose reads return canned data and whose writes are no-ops.
const stubRepository = (all: TupleKey[] = []): TupleRepository => ({
    has: () => false,
    findByObjectRelation: () => [],
    findAll: () => all,
    write: () => {},
    delete: () => {},
});

// ── Mocks (behaviour verification) ──────────────────────────────────────────

// A MOCK evaluator that records every request it is asked to evaluate.
const makeMockEvaluator = (result: CheckResult) => {
    const calls: CheckRequest[] = [];
    const evaluator: Evaluator = {
        evaluate: async (request) => {
            calls.push(request);
            return result;
        },
    };
    return { evaluator, calls };
};

// A MOCK repository that records write/delete calls and the filters passed to
// findAll.
const makeMockRepository = () => {
    const writes: TupleKey[] = [];
    const deletes: TupleKey[] = [];
    const findFilters: (TupleFilter | undefined)[] = [];
    const repository: TupleRepository = {
        has: () => false,
        findByObjectRelation: () => [],
        findAll: (filter) => {
            findFilters.push(filter);
            return [];
        },
        write: (t) => {
            writes.push(t);
        },
        delete: (t) => {
            deletes.push(t);
        },
    };
    return { repository, writes, deletes, findFilters };
};

const sampleRequest = (): CheckRequest => ({
    user: alice,
    relation: "can_edit",
    object: document("roadmapDocument"),
});

// ── check ───────────────────────────────────────────────────────────────────

describe("makeAuthzService — check", () => {
    it("returns the evaluator's result (state)", async () => {
        // Arrange: a STUB evaluator pinned to an allowed result.
        const domain = makeAuthzService({
            repository: stubRepository(),
            evaluator: stubEvaluator({ allowed: true, trace: ["Result: allowed"] }),
        });

        // Act
        const result = await domain.check(sampleRequest());

        // Assert
        expect(result.allowed).toBe(true);
    });

    it("propagates an evaluator failure (state)", async () => {
        // Arrange: a STUB evaluator that rejects.
        const error = new Error("evaluator exploded");
        const domain = makeAuthzService({
            repository: stubRepository(),
            evaluator: failingEvaluator(error),
        });

        // Act + Assert
        await expect(domain.check(sampleRequest())).rejects.toBe(error);
    });

    it("delegates the exact request to the evaluator (behaviour)", async () => {
        // Arrange: a MOCK evaluator so we can verify the delegation, not the result.
        const { evaluator, calls } = makeMockEvaluator({ allowed: true, trace: [] });
        const domain = makeAuthzService({ repository: stubRepository(), evaluator });
        const request = sampleRequest();

        // Act
        await domain.check(request);

        // Assert: exactly one delegation, with the request unchanged.
        expect(calls).toEqual([request]);
    });
});

// ── writeTuples / deleteTuples ──────────────────────────────────────────────

describe("makeAuthzService — writeTuples", () => {
    it("writes each tuple to the repository in order (behaviour)", async () => {
        // Arrange: a MOCK repository to capture the write interactions.
        const { repository, writes, deletes } = makeMockRepository();
        const domain = makeAuthzService({ repository, evaluator: stubEvaluator({ allowed: false, trace: [] }) });
        const tuples: TupleKey[] = [
            tuple(document("d1"), "owner", alice),
            tuple(document("d1"), "workspace", workspace("ws")),
        ];

        // Act
        await domain.writeTuples(tuples);

        // Assert
        expect(writes).toEqual(tuples);
        expect(deletes).toHaveLength(0);
    });

    it("does not touch the repository for an empty list (behaviour)", async () => {
        // Arrange
        const { repository, writes } = makeMockRepository();
        const domain = makeAuthzService({ repository, evaluator: stubEvaluator({ allowed: false, trace: [] }) });

        // Act
        await domain.writeTuples([]);

        // Assert
        expect(writes).toHaveLength(0);
    });
});

describe("makeAuthzService — deleteTuples", () => {
    it("deletes each tuple from the repository (behaviour)", async () => {
        // Arrange: a MOCK repository to capture the delete interactions.
        const { repository, writes, deletes } = makeMockRepository();
        const domain = makeAuthzService({ repository, evaluator: stubEvaluator({ allowed: false, trace: [] }) });
        const t = tuple(document("d1"), "owner", alice);

        // Act
        await domain.deleteTuples([t]);

        // Assert
        expect(deletes).toEqual([t]);
        expect(writes).toHaveLength(0);
    });
});

// ── listTuples ──────────────────────────────────────────────────────────────

describe("makeAuthzService — listTuples", () => {
    it("returns the repository's tuples (state)", async () => {
        // Arrange: a STUB repository with canned contents.
        const stored: TupleKey[] = [tuple(document("d1"), "owner", alice)];
        const domain = makeAuthzService({
            repository: stubRepository(stored),
            evaluator: stubEvaluator({ allowed: false, trace: [] }),
        });

        // Act
        const got = await domain.listTuples();

        // Assert
        expect(got).toEqual(stored);
    });

    it("passes the filter through to the repository (behaviour)", async () => {
        // Arrange: a MOCK repository so we can verify the filter is forwarded.
        const { repository, findFilters } = makeMockRepository();
        const domain = makeAuthzService({ repository, evaluator: stubEvaluator({ allowed: false, trace: [] }) });
        const filter: TupleFilter = { object: workspace("productWorkspace"), relation: "editor" };

        // Act
        await domain.listTuples(filter);

        // Assert
        expect(findFilters).toEqual([filter]);
    });
});
