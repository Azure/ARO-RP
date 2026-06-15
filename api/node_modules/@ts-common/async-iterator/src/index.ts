import * as sync from "@ts-common/iterator"

export type Entry<T> = sync.Entry<T>

export type AsyncIterableEx<T> = {
  readonly fold: <A>(func: (a: A, b: T, i: number) => Promise<A> | A, init: A) => Promise<A>
  readonly toArray: () => Promise<readonly T[]>
  readonly entries: () => AsyncIterableEx<Entry<T>>
  readonly map: <R>(func: (v: T, i: number) => Promise<R> | R) => AsyncIterableEx<R>
  readonly flatMap: <R>(func: (v: T, i: number) => AsyncIterable<R>) => AsyncIterableEx<R>
  readonly filter: (func: (v: T, i: number) => Promise<boolean> | boolean) => AsyncIterableEx<T>
} & AsyncIterable<T>

export const iterable = <T>(createIterator: () => AsyncIterator<T>): AsyncIterableEx<T> => {
  const property = <P extends readonly unknown[], R>(f: (self: AsyncIterable<T>, ...p: P) => R) =>
    (...p: P) => f(result, ...p)
  const result: AsyncIterableEx<T> = {
    [Symbol.asyncIterator]: createIterator,
    fold: property(fold),
    toArray: property(toArray),
    entries: property(entries),
    map: property(map),
    flatMap: property(flatMap),
    filter: property(filter),
  }
  return result
}

export const fromSync = <T>(input: sync.Iterable<T>): AsyncIterableEx<T> =>
  iterable(async function *(): AsyncIterator<T> { yield *input })

export const fromSequence = <T>(...a: readonly T[]): AsyncIterableEx<T> => fromSync(a)

export const fromPromise = <T>(p: Promise<sync.Iterable<T>>): AsyncIterableEx<T> =>
  iterable(async function *(): AsyncIterator<T> { yield *await p })

export const fold = async <T, A>(
  input: AsyncIterable<T> | undefined,
  func: (a: A, b: T, i: number) => A | Promise<A>,
  init: A,
): Promise<A> => {
  // tslint:disable-next-line:no-let
  let result: A = init
  /* tslint:disable-next-line:no-loop-statement */
  for await (const [index, value] of entries(input)) {
    /* tslint:disable-next-line:no-expression-statement */
    result = await func(result, value, index)
  }
  return result
}

export const toArray = <T>(input: AsyncIterable<T> | undefined): Promise<readonly T[]> =>
  fold(
    input,
    (a, i) => [...a, i],
    new Array<T>()
  )

export const entries = <T>(input: AsyncIterable<T> | undefined): AsyncIterableEx<Entry<T>> =>
  iterable(async function *(): AsyncIterator<sync.Entry<T>> {
    // tslint:disable-next-line:no-if-statement
    if (input === undefined) {
      return
    }
    // tslint:disable-next-line:no-let
    let index = 0
    // tslint:disable-next-line:no-loop-statement
    for await (const value of input) {
      yield [index, value]
      // tslint:disable-next-line:no-expression-statement
      index += 1
    }
  })

export const map = <T, I>(
  input: AsyncIterable<I> | undefined,
  func: (v: I, i: number) => Promise<T> | T,
): AsyncIterableEx<T> =>
  iterable(async function *(): AsyncIterator<T> {
    /* tslint:disable-next-line:no-loop-statement */
    for await (const [index, value] of entries(input)) {
      yield func(value, index)
    }
  })

export const flatten = <T>(input: AsyncIterable<AsyncIterable<T> | undefined> | undefined): AsyncIterableEx<T> =>
  iterable(async function *(): AsyncIterator<T> {
    // tslint:disable-next-line:no-if-statement
    if (input === undefined) {
      return
    }
    // tslint:disable-next-line:no-loop-statement
    for await (const v of input) {
      // tslint:disable-next-line:no-if-statement
      if (v !== undefined) {
        yield *v
      }
    }
  })

export const flatMap = <T, I>(
  input: AsyncIterable<I> | undefined,
  func: (v: I, i: number) => AsyncIterable<T> | undefined,
): AsyncIterableEx<T> =>
    flatten(map(input, func))

// tslint:disable-next-line:no-empty
export const empty = <T>(): AsyncIterableEx<T> => iterable(async function *(): AsyncIterator<T> {})

export const filter = <T>(
  input: AsyncIterable<T> | undefined,
  func: (v: T, i: number) => Promise<boolean> | boolean
): AsyncIterableEx<T> =>
  flatMap(
    input,
    (v, i) => iterable<T>(async function *() {
      // tslint:disable-next-line:no-if-statement
      if (await func(v, i)) {
        yield v
      }
    })
  )
