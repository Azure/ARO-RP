let manifest;
try {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    manifest = (await import("../manifest.js")).default;
}
catch {
    // Construct path dynamically so bundlers cannot statically resolve this import.
    // This fallback is only used when running directly from source during development.
    const name = ["../dist", "manifest.js"].join("/");
    manifest = (await import(/* @vite-ignore */ /* webpackIgnore: true */ name)).default;
}
export const typespecVersion = manifest.version;
export const MANIFEST = manifest;
//# sourceMappingURL=manifest.js.map