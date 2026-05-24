export const isJsonObject = (v: unknown): v is Record<string, unknown> =>
    typeof v === "object" && v !== null && !Array.isArray(v);

export const stringField = (obj: Record<string, unknown>, key: string): string => {
    if (typeof obj[key] !== "string" || obj[key] === "") {
        throw new Error(`Missing or empty field: ${key}`);
    }
    return obj[key];
};
