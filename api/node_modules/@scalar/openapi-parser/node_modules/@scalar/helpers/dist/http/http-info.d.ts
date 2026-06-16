export type HttpInfo = {
    short: string;
    colorClass: `text-${string}`;
    colorVar: `var(--scalar-color-${string})`;
    backgroundColor: string;
};
/**
 * HTTP methods in a specific order
 * Do not change the order
 */
export declare const REQUEST_METHODS: {
    readonly get: {
        readonly short: "GET";
        readonly colorClass: "text-blue";
        readonly colorVar: "var(--scalar-color-blue)";
        readonly backgroundColor: "bg-blue/10";
    };
    readonly post: {
        readonly short: "POST";
        readonly colorClass: "text-green";
        readonly colorVar: "var(--scalar-color-green)";
        readonly backgroundColor: "bg-green/10";
    };
    readonly put: {
        readonly short: "PUT";
        readonly colorClass: "text-orange";
        readonly colorVar: "var(--scalar-color-orange)";
        readonly backgroundColor: "bg-orange/10";
    };
    readonly patch: {
        readonly short: "PATCH";
        readonly colorClass: "text-yellow";
        readonly colorVar: "var(--scalar-color-yellow)";
        readonly backgroundColor: "bg-yellow/10";
    };
    readonly delete: {
        readonly short: "DEL";
        readonly colorClass: "text-red";
        readonly colorVar: "var(--scalar-color-red)";
        readonly backgroundColor: "bg-red/10";
    };
    readonly options: {
        readonly short: "OPTS";
        readonly colorClass: "text-purple";
        readonly colorVar: "var(--scalar-color-purple)";
        readonly backgroundColor: "bg-purple/10";
    };
    readonly head: {
        readonly short: "HEAD";
        readonly colorClass: "text-c-2";
        readonly colorVar: "var(--scalar-color-2)";
        readonly backgroundColor: "bg-c-2/10";
    };
    readonly trace: {
        readonly short: "TRACE";
        readonly colorClass: "text-c-2";
        readonly colorVar: "var(--scalar-color-2)";
        readonly backgroundColor: "bg-c-2/10";
    };
};
/**
 * Accepts an HTTP Method name and returns some properties for the tag
 */
export declare const getHttpMethodInfo: (methodName: string) => {
    readonly short: "GET";
    readonly colorClass: "text-blue";
    readonly colorVar: "var(--scalar-color-blue)";
    readonly backgroundColor: "bg-blue/10";
} | {
    readonly short: "POST";
    readonly colorClass: "text-green";
    readonly colorVar: "var(--scalar-color-green)";
    readonly backgroundColor: "bg-green/10";
} | {
    readonly short: "PUT";
    readonly colorClass: "text-orange";
    readonly colorVar: "var(--scalar-color-orange)";
    readonly backgroundColor: "bg-orange/10";
} | {
    readonly short: "PATCH";
    readonly colorClass: "text-yellow";
    readonly colorVar: "var(--scalar-color-yellow)";
    readonly backgroundColor: "bg-yellow/10";
} | {
    readonly short: "DEL";
    readonly colorClass: "text-red";
    readonly colorVar: "var(--scalar-color-red)";
    readonly backgroundColor: "bg-red/10";
} | {
    readonly short: "OPTS";
    readonly colorClass: "text-purple";
    readonly colorVar: "var(--scalar-color-purple)";
    readonly backgroundColor: "bg-purple/10";
} | {
    readonly short: "HEAD";
    readonly colorClass: "text-c-2";
    readonly colorVar: "var(--scalar-color-2)";
    readonly backgroundColor: "bg-c-2/10";
} | {
    readonly short: "TRACE";
    readonly colorClass: "text-c-2";
    readonly colorVar: "var(--scalar-color-2)";
    readonly backgroundColor: "bg-c-2/10";
};
//# sourceMappingURL=http-info.d.ts.map