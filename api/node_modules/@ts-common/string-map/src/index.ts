import * as _ from "@ts-common/iterator"

export const enum EntryIndex {
    Key = 0,
    Value = 1,
}

export type Entry<T> = readonly [string, T]

// export const entry: <T>(key: string, value: T) => Entry<T> = tuple2

export const entryKey = <T>(e: Entry<T>): string =>
    e[EntryIndex.Key]

export const entryValue = <T>(e: Entry<T>): T =>
    e[EntryIndex.Value]

export type PartialStringMap<K extends string, V> = {
    readonly [k in K]?: V
}

export interface StringMap<T> {
    readonly [k: string]: T | undefined
}

export const toStringMap = <K extends string, V>(v: PartialStringMap<K, V>): StringMap<V> => v

export interface MutableStringMap<T> {
    // tslint:disable-next-line:readonly-keyword
    [key: string]: T|undefined
}

export type StringMapItem<T> = T extends StringMap<infer I> ? I : never

export const allKeys = <T>(input: StringMap<T>|undefined|null): _.IterableEx<string> =>
    objectAllKeys<string, T>(input)

export const objectAllKeys = <K extends string, T>(input: PartialStringMap<K, T>|undefined|null): _.IterableEx<K> =>
    _.iterable(function *(): _.Iterator<K> {
        // tslint:disable-next-line:no-if-statement
        if (input === undefined || input === null) {
            return
        }
        // tslint:disable-next-line:no-loop-statement
        for (const key in input) {
            yield key
        }
    })

export const entries = <T>(input: StringMap<T>|undefined|null): _.IterableEx<Entry<T>> =>
    objectEntries<string, T>(input)

export const objectEntries = <K extends string, T>(
    input: PartialStringMap<K, T>|undefined|null
): _.IterableEx<readonly [K, T]> => {
    // tslint:disable-next-line:no-if-statement
    if (input === undefined || input === null) {
        return _.empty()
    }
    return objectAllKeys(input)
        .filterMap(key => {
            const value = input[key]
            return value !== undefined ? [key, value as T] as const : undefined
        })
}

export const keys = <T>(input: StringMap<T>|undefined|null): _.IterableEx<string> =>
    entries(input).map(entryKey)

export const values = <T>(input: StringMap<T>|undefined|null): _.IterableEx<T> =>
    entries(input).map(entryValue)

export const groupBy = <T>(
    input: _.Iterable<Entry<T>>,
    reduceFunc: (a: T, b: T) => T
): StringMap<T> => {
    /* tslint:disable-next-line:readonly-keyword */
    const result: MutableStringMap<T> = {}
    _.forEach(input, ([key, value]) => {
        const prior = result[key]
        /* tslint:disable-next-line:no-object-mutation no-expression-statement */
        result[key] = prior === undefined ? value : reduceFunc(prior, value)
    })
    return result
}

export const stringMap = <T>(input: _.Iterable<Entry<T>>): StringMap<T> =>
    // tslint:disable-next-line:variable-name
    groupBy(input, (_a, b) => b)

export const map = <S, R>(source: StringMap<S>, f: (v: S, k: string) => R): StringMap<R> =>
    stringMap(entries(source).map(([k, v]) => [k, f(v, k)] as const))

export const merge = <T>(...a: readonly (StringMap<T>|undefined)[]): StringMap<T> =>
    stringMap(_.map(a, entries).flatMap(v => v))

// Performs a partial deep comparison between object and source to determine if object contains
// equivalent property values.
// See also https://lodash.com/docs/4.17.10#isMatch
export const isMatch = <O, S>(object: StringMap<O>, source: StringMap<S>): boolean =>
    entries(source).every(([key, value]) => _.isStrictEqual(object[key], value))

export const isEqual = <A, B>(a: StringMap<A>, b: StringMap<B>): boolean =>
    _.isStrictEqual(a, b) || (isMatch(a, b) && isMatch(b, a))

export const isEmpty = <T>(a: StringMap<T>): boolean =>
    entries(a).isEmpty()
