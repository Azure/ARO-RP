export default class BaseValidationError {
    constructor(options: {
        isIdentifierLocation: boolean;
    }, { data, schema, jsonAst, jsonRaw }: {
        data: any;
        schema: any;
        jsonAst: any;
        jsonRaw: any;
    });
    options: {
        isIdentifierLocation: boolean;
    };
    data: any;
    schema: any;
    jsonAst: any;
    jsonRaw: any;
    /**
     * @return {string}
     */
    get instancePath(): string;
    getError(): void;
}
//# sourceMappingURL=base.d.ts.map