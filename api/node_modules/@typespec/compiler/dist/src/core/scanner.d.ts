import { DiagnosticHandler } from "./diagnostics.js";
import { SourceFile, TextRange, TypeSpecScriptNode } from "./types.js";
export declare enum Token {
    None = 0,
    Invalid = 1,
    EndOfFile = 2,
    Identifier = 3,
    NumericLiteral = 4,
    StringLiteral = 5,
    StringTemplateHead = 6,
    StringTemplateMiddle = 7,
    StringTemplateTail = 8,
    SingleLineComment = 9,
    MultiLineComment = 10,
    NewLine = 11,
    Whitespace = 12,
    ConflictMarker = 13,
    DocText = 14,
    DocCodeSpan = 15,
    DocCodeFenceDelimiter = 16,
    OpenBrace = 17,
    CloseBrace = 18,
    OpenParen = 19,
    CloseParen = 20,
    OpenBracket = 21,
    CloseBracket = 22,
    Dot = 23,
    Ellipsis = 24,
    Semicolon = 25,
    Comma = 26,
    LessThan = 27,
    GreaterThan = 28,
    Equals = 29,
    Ampersand = 30,
    Bar = 31,
    Question = 32,
    Colon = 33,
    ColonColon = 34,
    At = 35,
    AtAt = 36,
    Hash = 37,
    HashBrace = 38,
    HashBracket = 39,
    Star = 40,
    ForwardSlash = 41,
    Plus = 42,
    Hyphen = 43,
    Exclamation = 44,
    LessThanEquals = 45,
    GreaterThanEquals = 46,
    AmpsersandAmpersand = 47,
    BarBar = 48,
    EqualsEquals = 49,
    ExclamationEquals = 50,
    EqualsGreaterThan = 51,
    ImportKeyword = 52,
    ModelKeyword = 53,
    ScalarKeyword = 54,
    NamespaceKeyword = 55,
    UsingKeyword = 56,
    OpKeyword = 57,
    EnumKeyword = 58,
    AliasKeyword = 59,
    IsKeyword = 60,
    InterfaceKeyword = 61,
    UnionKeyword = 62,
    ProjectionKeyword = 63,
    ElseKeyword = 64,
    IfKeyword = 65,
    DecKeyword = 66,
    ConstKeyword = 67,
    InitKeyword = 68,
    ExternKeyword = 69,
    InternalKeyword = 70,
    ExtendsKeyword = 71,
    FnKeyword = 72,
    TrueKeyword = 73,
    FalseKeyword = 74,
    ReturnKeyword = 75,
    VoidKeyword = 76,
    NeverKeyword = 77,
    UnknownKeyword = 78,
    ValueOfKeyword = 79,
    TypeOfKeyword = 80,
    StatemachineKeyword = 81,
    MacroKeyword = 82,
    PackageKeyword = 83,
    MetadataKeyword = 84,
    EnvKeyword = 85,
    ArgKeyword = 86,
    DeclareKeyword = 87,
    ArrayKeyword = 88,
    StructKeyword = 89,
    RecordKeyword = 90,
    ModuleKeyword = 91,
    ModKeyword = 92,
    SymKeyword = 93,
    ContextKeyword = 94,
    PropKeyword = 95,
    PropertyKeyword = 96,
    ScenarioKeyword = 97,
    PubKeyword = 98,
    SubKeyword = 99,
    TypeRefKeyword = 100,
    TraitKeyword = 101,
    ThisKeyword = 102,
    SelfKeyword = 103,
    SuperKeyword = 104,
    KeyofKeyword = 105,
    WithKeyword = 106,
    ImplementsKeyword = 107,
    ImplKeyword = 108,
    SatisfiesKeyword = 109,
    FlagKeyword = 110,
    AutoKeyword = 111,
    PartialKeyword = 112,
    PrivateKeyword = 113,
    PublicKeyword = 114,
    ProtectedKeyword = 115,
    SealedKeyword = 116,
    LocalKeyword = 117,
    AsyncKeyword = 118
}
export type DocToken = Token.NewLine | Token.Whitespace | Token.ConflictMarker | Token.Star | Token.At | Token.CloseBrace | Token.Identifier | Token.Hyphen | Token.DocText | Token.DocCodeSpan | Token.DocCodeFenceDelimiter | Token.EndOfFile;
export type StringTemplateToken = Token.StringTemplateHead | Token.StringTemplateMiddle | Token.StringTemplateTail;
export interface Scanner {
    /** The source code being scanned. */
    readonly file: SourceFile;
    /** The offset in UTF-16 code units to the current position at the start of the next token. */
    readonly position: number;
    /** The current token */
    readonly token: Token;
    /** The offset in UTF-16 code units to the start of the current token. */
    readonly tokenPosition: number;
    /** The flags on the current token. */
    readonly tokenFlags: TokenFlags;
    /** Advance one token. */
    scan(): Token;
    /** Advance one token inside DocComment. Use inside {@link scanRange} callback over DocComment range. */
    scanDoc(): DocToken;
    /**
     * Unconditionally back up and scan a template expression portion.
     * @param tokenFlags Token Flags for head StringTemplateToken
     */
    reScanStringTemplate(tokenFlags: TokenFlags): StringTemplateToken;
    /**
     * Finds the indent for the given triple quoted string.
     * @param start
     * @param end
     */
    findTripleQuotedStringIndent(start: number, end: number): [number, number];
    /**
     * Unindent and unescape the triple quoted string rawText
     */
    unindentAndUnescapeTripleQuotedString(start: number, end: number, indentationStart: number, indentationEnd: number, token: Token.StringLiteral | StringTemplateToken, tokenFlags: TokenFlags): string;
    /** Reset the scanner to the given start and end positions, invoke the callback, and then restore scanner state. */
    scanRange<T>(range: TextRange, callback: () => T): T;
    /** Determine if the scanner has reached the end of the input. */
    eof(): boolean;
    /** The exact spelling of the current token. */
    getTokenText(): string;
    /**
     * The value of the current token.
     *
     * String literals are escaped and unquoted, identifiers are normalized,
     * and all other tokens return their exact spelling sames as
     * getTokenText().
     */
    getTokenValue(): string;
}
export declare enum TokenFlags {
    None = 0,
    Escaped = 1,
    TripleQuoted = 2,
    Unterminated = 4,
    NonAscii = 8,
    DocComment = 16,
    Backticked = 32
}
export declare function isTrivia(token: Token): boolean;
export declare function isComment(token: Token): boolean;
export declare function isKeyword(token: Token): boolean;
/** If is a keyword with no actual use right now but will be in the future. */
export declare function isReservedKeyword(token: Token): boolean;
export declare function isPunctuation(token: Token): boolean;
export declare function isModifier(token: Token): boolean;
export declare function isStatementKeyword(token: Token): boolean;
export declare function createScanner(source: string | SourceFile, diagnosticHandler: DiagnosticHandler): Scanner;
/**
 *
 * @param script
 * @param position
 * @param endPosition exclude
 * @returns return === endPosition (or -1) means not found non-trivia until endPosition + 1
 */
export declare function skipTriviaBackward(script: TypeSpecScriptNode, position: number, endPosition?: number): number;
/**
 *
 * @param input
 * @param position
 * @param endPosition exclude
 * @returns return === endPosition (or input.length) means not found non-trivia until endPosition - 1
 */
export declare function skipTrivia(input: string, position: number, endPosition?: number): number;
export declare function skipWhiteSpace(input: string, position: number, endPosition?: number): number;
export declare function skipContinuousIdentifier(input: string, position: number, isBackward?: boolean): number;
//# sourceMappingURL=scanner.d.ts.map