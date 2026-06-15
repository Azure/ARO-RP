/** Create a {@link ResolveModuleHost} from a {@link SystemHost}. */
export function createResolveModuleHost(host) {
    return {
        realpath: host.realpath,
        stat: host.stat,
        readFile: async (path) => {
            const file = await host.readFile(path);
            return file.text;
        },
    };
}
//# sourceMappingURL=module-host.js.map