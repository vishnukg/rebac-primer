// Parses a TCP port from an environment-variable string, falling back to a
// default when unset/blank. Shared by the service composition roots (authz,
// documents) so the parsing/validation lives in one place.

const readPort = (value: string | undefined, fallback: number): number => {
    if (!value?.trim()) return fallback;
    const p = Number(value);
    if (!Number.isInteger(p) || p < 1 || p > 65_535) throw new Error(`Invalid port: ${value}`);
    return p;
};

export default readPort;
