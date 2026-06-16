export class TestHostError extends Error {
    code;
    constructor(message, code) {
        super(message);
        this.code = code;
    }
}
// #endregion
//# sourceMappingURL=types.js.map