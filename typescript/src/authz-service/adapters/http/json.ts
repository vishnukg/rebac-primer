export const isJsonObject = (v: unknown): v is Record<string, unknown> =>
    typeof v === "object" && v !== null && !Array.isArray(v);

export const stringField = (obj: Record<string, unknown>, key: string): string => {
    if (typeof obj[key] !== "string" || obj[key] === "") {
        throw new Error(`Missing or empty field: ${key}`);
    }
    return obj[key];
};

export const optionalStringField = (
    source: URLSearchParams | Record<string, unknown>,
    key: string,
): string | undefined => {
    const v = source instanceof URLSearchParams ? source.get(key) : source[key];
    return typeof v === "string" && v !== "" ? v : undefined;
};
