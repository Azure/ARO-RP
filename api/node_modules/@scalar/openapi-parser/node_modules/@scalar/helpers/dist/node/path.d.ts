export declare function resolve(...parameters: Array<string>): string;
export declare function normalize(inputPath: string): string;
export declare function isAbsolute(path: string): boolean;
export declare function join(...paths: string[]): string;
export declare function relative(from: string, to: string): string;
export declare const sep = "/";
export declare const delimiter = ":";
export declare function dirname(path: string): string;
export declare function basename(path: string, ext?: string): string;
export declare function extname(path: string): string;
export declare const path: {
    extname: typeof extname;
    basename: typeof basename;
    dirname: typeof dirname;
    sep: string;
    delimiter: string;
    relative: typeof relative;
    join: typeof join;
    isAbsolute: typeof isAbsolute;
    normalize: typeof normalize;
    resolve: typeof resolve;
};
//# sourceMappingURL=path.d.ts.map