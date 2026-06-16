import type { ArrayValue, BooleanLiteral, BooleanValue, Entity, Enum, EnumMember, EnumValue, Interface, Model, ModelProperty, Namespace, NumericLiteral, NumericValue, ObjectValue, Operation, Scalar, ScalarValue, StringLiteral, StringValue, Type, Union, UnionVariant, Value } from "../core/types.js";
export type Marker<T extends Entity, N extends string> = T extends Type ? TypeMarker<T, N> : T extends Value ? ValueMarker<T, N> : never;
export interface TypeMarker<T extends Type, N extends string> {
    readonly entityKind: "Type";
    readonly kind?: T["kind"];
    readonly name: N;
}
export interface ValueMarker<T extends Value, N extends string> {
    readonly entityKind: "Value";
    readonly valueKind?: T["valueKind"];
    readonly name: N;
}
export type MarkerConfig<T extends Record<string, Entity>> = {
    [K in keyof T]: Marker<T[K], K & string>;
};
export interface TemplateWithMarkers<T extends Record<string, Entity>> {
    readonly isTemplateWithMarkers: true;
    readonly code: string;
    readonly markers: MarkerConfig<T>;
}
export declare const TemplateWithMarkers: {
    is: (value: unknown) => value is TemplateWithMarkers<any>;
};
/** Specify that this value is dynamic and needs to be interpolated with the given keys */
declare function code<const T extends (Marker<Entity, string> | string)[]>(strings: TemplateStringsArray, ...keys: T): TemplateWithMarkers<Prettify<CollectType<T>>>;
/** TypeSpec template marker */
export declare const t: {
    /**
     * Define a marked code block
     *
     * @example
     * ```ts
     * const code = t.code`model ${t.model("Foo")} { bar: string }`;
     * ```
     */
    code: typeof code;
    /** Mark any type */
    type: <const N extends string>(name: N) => TypeMarker<Type, N>;
    /** Mark a model */
    model: <const N extends string>(name: N) => TypeMarker<Model, N>;
    /** Mark an enum */
    enum: <const N extends string>(name: N) => TypeMarker<Enum, N>;
    /** Mark an union */
    union: <const N extends string>(name: N) => TypeMarker<Union, N>;
    /** Mark an interface */
    interface: <const N extends string>(name: N) => TypeMarker<Interface, N>;
    /** Mark an operation */
    op: <const N extends string>(name: N) => TypeMarker<Operation, N>;
    /** Mark an enum member */
    enumMember: <const N extends string>(name: N) => TypeMarker<EnumMember, N>;
    /** Mark a model property */
    modelProperty: <const N extends string>(name: N) => TypeMarker<ModelProperty, N>;
    /** Mark a namespace */
    namespace: <const N extends string>(name: N) => TypeMarker<Namespace, N>;
    /** Mark a scalar */
    scalar: <const N extends string>(name: N) => TypeMarker<Scalar, N>;
    /** Mark a union variant */
    unionVariant: <const N extends string>(name: N) => TypeMarker<UnionVariant, N>;
    /** Mark a boolean literal */
    boolean: <const N extends string>(name: N) => TypeMarker<BooleanLiteral, N>;
    /** Mark a number literal */
    number: <const N extends string>(name: N) => TypeMarker<NumericLiteral, N>;
    /** Mark a string literal */
    string: <const N extends string>(name: N) => TypeMarker<StringLiteral, N>;
    /** Mark any value */
    value: <const N extends string>(name: N) => ValueMarker<Value, N>;
    /** Mark an object value */
    object: <const N extends string>(name: N) => ValueMarker<ObjectValue, N>;
    /** Mark an array value */
    array: <const N extends string>(name: N) => ValueMarker<ArrayValue, N>;
    /** Mark a numeric value */
    numericValue: <const N extends string>(name: N) => ValueMarker<NumericValue, N>;
    /** Mark a string value */
    stringValue: <const N extends string>(name: N) => ValueMarker<StringValue, N>;
    /** Mark a boolean value */
    booleanValue: <const N extends string>(name: N) => ValueMarker<BooleanValue, N>;
    /** Mark a scalar value */
    scalarValue: <const N extends string>(name: N) => ValueMarker<ScalarValue, N>;
    /** Mark an enum value */
    enumValue: <const N extends string>(name: N) => ValueMarker<EnumValue, N>;
};
type Prettify<T extends Record<string, Entity>> = {
    [K in keyof T]: T[K] & Entity;
} & {};
type InferType<T> = T extends Marker<infer K, infer _> ? K : never;
type CollectType<T extends ReadonlyArray<Marker<Entity, string> | string>> = {
    [K in T[number] as K extends Marker<infer _K, infer N> ? N : never]: InferType<K>;
};
type UnionToIntersection<U> = (U extends any ? (k: U) => void : never) extends (k: infer I) => void ? I : never;
type FlattenRecord<T extends Record<string, unknown>> = UnionToIntersection<T[keyof T]>;
type FlattenTemplates<M extends Record<string, string | TemplateWithMarkers<any>>> = FlattenRecord<{
    [K in keyof M]: M[K] extends TemplateWithMarkers<infer T> ? T : never;
}>;
export type GetMarkedEntities<M extends string | TemplateWithMarkers<any> | Record<string, string | TemplateWithMarkers<any>>> = M extends Record<string, string | TemplateWithMarkers<any>> ? FlattenTemplates<M> : M extends string | TemplateWithMarkers<infer R> ? R : never;
export {};
//# sourceMappingURL=marked-template.d.ts.map