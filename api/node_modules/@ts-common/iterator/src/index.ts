/**
 * See this PR https://github.com/microsoft/TypeScript/pull/30790
 */
export type IteratorResult<T> = {
    /**
     * - Has the value `true` if the iterator is past the end of the iterated sequence. In this case value optionally
     *   specifies the return value of the iterator.
     * - Has the value `false` if the iterator was able to produce the next value in the sequence. This is equivalent of
     *   not specifying the done property altogether.
     */
    readonly done: boolean;
    /**
     * any JavaScript value returned by the iterator. Can be omitted when done is true.
     */
    readonly value: T;
}

export type Iterator<T> = {
    /**
     * Returns `IterableResult<T>`.
     */
    readonly next: () => IteratorResult<T>;
}

export type Iterable<T> = {
    /**
     * The function returns an iterator.
     */
    readonly [Symbol.iterator]: () => Iterator<T>;
}

export type IterableEx<T> = Iterable<T> & {
    /**
     * The function returns an iterator of a this container own enumerable number-keyed value [key, value] pairs.
     */
    readonly entries: () => IterableEx<Entry<T>>
    /**
     * Creates a new sequence whose values are calculated by passing this sequence's elements through the given
     * function.
     */
    readonly map: <R>(func: (v: T, i: number) => R) => IterableEx<R>
    /**
     *  Returns the first element of this sequence or `undefined` if the sequence is empty.
     */
    readonly first: () => T | undefined
    /**
     * The flatMap() method first maps each element using a mapping function, then flattens the result.
     */
    readonly flatMap: <R>(func: (v: T, i: number) => Iterable<R>) => IterableEx<R>
    /**
     * Creates a new sequence whose values are the elements of this sequence which satisfy the specified predicate.
     */
    readonly filter: (func: (v: T, i: number) => boolean) => IterableEx<T>
    /**
     * Creates a new sequence whose values are calculated by passing this sequence's elements through the given
     * function. If the function `func` returns `undefined`, the item is removed from the sequence.
     */
    readonly filterMap: <R>(func: (v: T, i: number) => R | undefined) => IterableEx<R>
    /**
     * The forEach() method executes a provided function once for each sequence element.
     */
    readonly forEach: (func: (v: T, i: number) => void) => void
    /**
     * Creates a slice of sequence with n elements dropped from the beginning.
     */
    readonly drop: (n?: number) => IterableEx<T>
    /**
     * Creates a new sequence with all of the elements of this one, plus those of the given sequence(s).
     */
    readonly concat: (...input: readonly (Iterable<T> | undefined)[]) => IterableEx<T>
    /**
     * Creates a new sequence comprising the elements from the head of this sequence that satisfy some predicate. Once
     * an element is encountered that doesn't satisfy the predicate, iteration will stop.
     */
    readonly takeWhile: (func: (v: T, i: number) => boolean) => IterableEx<T>
    /**
     * Creates a slice of sequence with n elements taken from the beginning.
     */
    readonly take: (n?: number) => IterableEx<T>
    /**
     * This method is like find except that it returns the `Entry<T>` of the first element predicate returns truthy for
     * instead of the element itself. This is useful if the sequence can contain `undefined` values.
     */
    readonly findEntry: (func: (v: T, i: number) => boolean) => Entry<T> | undefined
    /**
     * Searches for the first element in the sequence satisfying a given predicate.
     */
    readonly find: (func: (v: T, i: number) => boolean) => T | undefined
    /**
     * Reduces collection to a value which is the accumulated result of running each element in collection thru
     * `func`, where each successive invocation is supplied the return value of the previous.
     */
    readonly fold: <A>(func: (a: A, b: T, i: number) => A, init: A) => A
    /**
     * Reduces collection to a value which is the accumulated result of running each element in collection thru
     * `func`, where each successive invocation is supplied the return value of the previous.
     *
     * The first element of collection is used as the initial value.
     */
    readonly reduce: (func: (a: T, b: T, i: number) => T) => T | undefined
    /**
     *  Returns the last element of this sequence or `undefined` if the sequence is empty.
     */
    readonly last: () => T | undefined
    /**
     * Checks whether at least one element in this sequence satisfies a given predicate (or, if no predicate is
     * specified, whether the sequence contains at least one element).
     */
    readonly some: (func?: (v: T, i: number) => boolean) => boolean
    /**
     * Checks whether every element in this sequence satisfies a given predicate.
     */
    readonly every: (func: (v: T, i: number) => boolean) => boolean
    /**
     * Creates a new sequence by combining the elements from this sequence with corresponding elements from the
     * specified sequence(s).
     */
    readonly zip: (...inputs: readonly (Iterable<T> | undefined)[]) => IterableEx<readonly T[]>
    /**
     * Checks if all items in the sequence are equal to the items in the given sequence `b`.
     */
    readonly isEqual: <B>(b: Iterable<B> | undefined, e?: (ai: T, bi: B) => boolean) => boolean
    /**
     * Creates an array snapshot of a sequence.
     */
    readonly toArray: () => readonly T[]
    /**
     * Creates a new array with the same elements as this sequence, but in the opposite order.
     */
    readonly reverse: () => readonly T[]
    /**
     * Checks whether the sequence has no elements.
     */
    readonly isEmpty: () => boolean
    /**
     * Creates a new sequence with every unique element from this one appearing exactly once (i.e., with duplicates
     * removed).
     */
    readonly uniq: (key?: (v: T) => unknown) => IterableEx<T>,
    /**
     * Creates a new sequence of accumulated values. It's exclusive scan so it always returns at least one value.
     */
    readonly scan: <A>(func: (a: A, b: T, i: number) => A, init: A) => IterableEx<A>,
    /**
     * Firstly, the function maps a state and each element using the `func` function, then flattens the result.
     */
    readonly flatScan: <A, R>(func: (a: A, b: T, i: number) => readonly [A, Iterable<R>], init: A) => IterableEx<R>,
}

export const iterable = <T>(createIterator: () => Iterator<T>): IterableEx<T> => {
    const it = { [Symbol.iterator]: createIterator }
    const property = <P extends readonly unknown[], R>(f: (self: Iterable<T>, ...p: P) => R) =>
        (...p: P) => f(it, ...p)
    return {
        [Symbol.iterator]: createIterator,
        concat: property(concat),
        drop: property(drop),
        entries: property(entries),
        every: property(every),
        filter: property(filter),
        filterMap: property(filterMap),
        find: property(find),
        findEntry: property(findEntry),
        flatMap: property(flatMap),
        fold: property(fold),
        forEach: property(forEach),
        isEmpty: property(isEmpty),
        isEqual: property(isEqual),
        last: property(last),
        map: property(map),
        reduce: property(reduce),
        reverse: property(reverse),
        some: property(some),
        take: property(take),
        takeWhile: property(takeWhile),
        toArray: property(toArray),
        uniq: property(uniq),
        zip: property(zip),
        scan: property(scan),
        flatScan: property(flatScan),
        first: property(first)
    }
}

export type Entry<T> = readonly [number, T]

export const ENTRY_KEY = 0
export const ENTRY_VALUE = 1

export const chain = <T>(input: readonly T[]): IterableEx<T> => iterable(() => input[Symbol.iterator]())

export const entries = <T>(input: Iterable<T> | undefined): IterableEx<Entry<T>> =>
    iterable(function *() {
        // tslint:disable-next-line:no-if-statement
        if (input === undefined) {
            return
        }
        let index = 0
        // tslint:disable-next-line:no-loop-statement
        for (const value of input) {
            yield [index, value] as const
            // tslint:disable-next-line:no-expression-statement
            index += 1
        }
    })

export const map = <T, I>(
    input: Iterable<I> | undefined,
    func: (v: I, i: number) => T,
): IterableEx<T> =>
    iterable(function *(): Iterator<T> {
        // tslint:disable-next-line:no-loop-statement
        for (const [index, value] of entries(input)) {
            yield func(value, index)
        }
    })

export const drop = <T>(input: Iterable<T> | undefined, n: number = 1): IterableEx<T> =>
    filter(input, (_, i) => n <= i)

export const flat = <T>(input: Iterable<Iterable<T> | undefined> | undefined): IterableEx<T> =>
    iterable(function *(): Iterator<T> {
        // tslint:disable-next-line:no-if-statement
        if (input === undefined) {
            return
        }
        // tslint:disable-next-line:no-loop-statement
        for (const v of input) {
            // tslint:disable-next-line:no-if-statement
            if (v !== undefined) {
                yield *v
            }
        }
    })

export const concat = <T>(...input: readonly (Iterable<T> | undefined)[]): IterableEx<T> =>
    flat(input)

export const takeWhile = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean,
): IterableEx<T> =>
    iterable(function *(): Iterator<T> {
        // tslint:disable-next-line:no-loop-statement
        for (const [index, value] of entries(input)) {
            // tslint:disable-next-line:no-if-statement
            if (!func(value, index)) {
                return
            }
            yield value
        }
    })

export const take = <T>(input: Iterable<T> | undefined, n: number = 1) =>
    takeWhile(input, (_, i) => i < n)

export const findEntry = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean,
): Entry<T> | undefined => {
    // tslint:disable-next-line:no-loop-statement
    for (const e of entries(input)) {
        // tslint:disable-next-line:no-if-statement
        if (func(e[ENTRY_VALUE], e[ENTRY_KEY])) {
            return e
        }
    }
    return undefined
}

export const find = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean,
): T | undefined => {
    const e = findEntry(input, func)
    return e === undefined ? undefined : e[ENTRY_VALUE]
}

export const flatMap = <T, I>(
    input: Iterable<I> | undefined,
    func: (v: I, i: number) => Iterable<T>,
): IterableEx<T> =>
    flat(map(input, func))

export const optionalToArray = <T>(v: T | undefined): readonly T[] =>
    v === undefined ? [] : [v]

export const filterMap = <T, I>(
    input: Iterable<I> | undefined,
    func: (v: I, i: number) => T | undefined,
): IterableEx<T> =>
    flatMap(input, (v, i) => optionalToArray(func(v, i)))

export const filter = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean,
): IterableEx<T> =>
    flatMap(input, (v, i) => func(v, i) ? [v] : [])

const infinite = (): IterableEx<void> =>
    iterable(function *(): Iterator<void> {
        // tslint:disable-next-line:no-loop-statement
        while (true) { yield }
    })

export const generate = <T>(func: (i: number) => T, count?: number): IterableEx<T> =>
    infinite()
        .takeWhile((_, i) => i !== count)
        .map((_, i) => func(i))

export const repeat = <T>(v: T, count?: number): IterableEx<T> =>
    generate(() => v, count)

export const scan = <T, A>(
    input: Iterable<T> | undefined,
    func: (a: A, b: T, i: number) => A,
    init: A,
): IterableEx<A> =>
    iterable(function *() {
        let result: A = init
        yield result
        // tslint:disable-next-line:no-loop-statement
        for (const [index, value] of entries(input)) {
            // tslint:disable-next-line:no-expression-statement
            result = func(result, value, index)
            yield result
        }
    })

export const flatScan = <T, A, R>(
    input: Iterable<T> | undefined,
    func: (a: A, b: T, i: number) => readonly [A, Iterable<R>],
    init: A
): IterableEx<R> =>
    iterable(function *() {
        let state = init
        // tslint:disable-next-line:no-loop-statement
        for (const [index, value] of entries(input)) {
            // tslint:disable-next-line:no-expression-statement
            const [newState, result] = func(state, value, index)
            // tslint:disable-next-line:no-expression-statement
            state = newState
            yield *result
        }
    })

export const fold = <T, A>(
    input: Iterable<T> | undefined,
    func: (a: A, b: T, i: number) => A,
    init: A,
): A => {
    let result: A = init
    // tslint:disable-next-line:no-loop-statement
    for (const [index, value] of entries(input)) {
        // tslint:disable-next-line:no-expression-statement
        result = func(result, value, index)
    }
    return result
}

export const reduce = <T>(
    input: Iterable<T> | undefined,
    func: (a: T, b: T, i: number) => T,
): T | undefined =>
    fold<T, T | undefined>(
        input,
        (a, b, i) => a !== undefined ? func(a, b, i) : b,
        undefined,
    )

export const first = <T>(input: Iterable<T> | undefined): T | undefined => {
    // tslint:disable-next-line:no-if-statement
    if (input !== undefined) {
        // tslint:disable-next-line:no-loop-statement
        for (const v of input) {
            return v
        }
    }
    return undefined
}

export const last = <T>(input: Iterable<T> | undefined): T | undefined =>
    reduce(input, (_, v) => v)

export const some = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean = () => true,
): boolean =>
    findEntry(input, func) !== undefined

export const every = <T>(
    input: Iterable<T> | undefined,
    func: (v: T, i: number) => boolean,
): boolean =>
    !some(input, (v, i) => !func(v, i))

export const forEach = <T>(input: Iterable<T> | undefined, func: (v: T, i: number) => void): void => {
    // tslint:disable-next-line:no-expression-statement
    fold<T, void>(input, (_, v, i) => { func(v, i) }, undefined)
}

export const sum = (input: Iterable<number> | undefined): number =>
    fold(input, (a, b) => a + b, 0)

export const min = (input: Iterable<number> | undefined): number =>
    fold(input, (a, b) => Math.min(a, b), Infinity)

export const max = (input: Iterable<number> | undefined): number =>
    fold(input, (a, b) => Math.max(a, b), -Infinity)

export const zip = <T>(...inputs: readonly (Iterable<T> | undefined)[]): IterableEx<readonly T[]> =>
    iterable(function *(): Iterator<readonly T[]> {
        const iterators = inputs.map(
            i => i === undefined ? [][Symbol.iterator]() : i[Symbol.iterator](),
        )
        // tslint:disable-next-line:no-loop-statement
        while (true) {
            const result = new Array<T>(inputs.length)
            // tslint:disable-next-line:no-loop-statement
            for (const [index, it] of entries(iterators)) {
                const v = it.next()
                // tslint:disable-next-line:no-if-statement
                if (v.done) {
                    return
                }
                // tslint:disable-next-line:no-object-mutation no-expression-statement
                result[index] = v.value
            }
            yield result
        }
    })

// TypeScript gives an error in case if type of a and type of b are different
export const isStrictEqual = (a: unknown, b: unknown) => a === b

export const isEqual = <A, B>(
    a: Iterable<A> | undefined,
    b: Iterable<B> | undefined,
    e: (ai: A, bi: B) => boolean = isStrictEqual,
): boolean => {
    // tslint:disable-next-line:no-if-statement
    if (isStrictEqual(a, b)) {
        return true
    }
    // tslint:disable-next-line:no-if-statement
    if (a === undefined || b === undefined) {
        return false
    }
    const ai = a[Symbol.iterator]()
    const bi = b[Symbol.iterator]()
    // tslint:disable-next-line:no-loop-statement
    while (true) {
        const av = ai.next()
        const bv = bi.next()
        // tslint:disable-next-line:no-if-statement
        if (av.done || bv.done) {
            return av.done === bv.done
        }
        // tslint:disable-next-line:no-if-statement
        if (!e(av.value, bv.value)) {
            return false
        }
    }
}

export const isArray = <T, U>(v: readonly T[] | U): v is readonly T[] =>
    v instanceof Array

export const toArray = <T>(i: Iterable<T> | undefined): readonly T[] =>
    i === undefined ? [] : Array.from(i)

export const reverse = <T>(i: Iterable<T> | undefined): readonly T[] =>
    fold(i, (a, b) => [b, ...a], new Array<T>())

export const arrayReverse = <T>(a: readonly T[]): IterableEx<T> =>
    iterable(function *() {
        // tslint:disable-next-line:no-loop-statement
        for (let i = a.length; i > 0;) {
            // tslint:disable-next-line:no-expression-statement
            i -= 1
            yield a[i]
        }
    })

export const isEmpty = <T>(i: Iterable<T> | undefined): boolean =>
    !some(i, () => true)

export const join = (i: Iterable<string> | undefined, separator: string): string => {
    const result = reduce(i, (a, b) => a + separator + b)
    return result === undefined ? "" : result
}

// tslint:disable-next-line:no-empty
export const empty = <T>() => iterable(function *(): Iterator<T> { })

export const dropRight = <T>(i: readonly T[] | undefined, n: number = 1): IterableEx<T> =>
    i === undefined ? empty() : take(i, i.length - n)

export const uniq = <T>(i: Iterable<T>, key: (v: T) => unknown = v => v): IterableEx<T> =>
    flatScan(
        i,
        (set, v: T): readonly [Set<unknown>, readonly T[]] => {
            const k = key(v)
            // tslint:disable-next-line:no-if-statement
            if (!set.has(k)) {
                // tslint:disable-next-line:no-expression-statement
                set.add(k)
                return [set, [v]]
            }
            return [set, []]
        },
        new Set<unknown>()
    )
