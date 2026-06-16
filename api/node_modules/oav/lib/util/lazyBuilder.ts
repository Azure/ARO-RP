interface BuilderListener<T> {
  resolve: (result: T) => void;
  reject: (err: any) => void;
}

export const getLazyBuilder = <SourceType, Key extends keyof SourceType & string>(
  key: Key,
  builder: (source: SourceType) => Promise<Exclude<SourceType[Key], undefined>>
) => {
  const errKey = "_err_" + key;
  const listeners = new Map<SourceType, Array<BuilderListener<SourceType[Key]>>>();

  return async (source: SourceType): Promise<Exclude<SourceType[Key], undefined>> => {
    let val = source[key];
    if (val !== undefined) {
      return val as any;
    }
    const err = (source as any)[errKey];
    if (err !== undefined) {
      throw err;
    }

    let listener = listeners.get(source);
    if (listener !== undefined) {
      return new Promise<any>((resolve, reject) => listener!.push({ resolve, reject }));
    }

    listener = [];
    listeners.set(source, listener);

    try {
      val = await builder(source);
      // eslint-disable-next-line require-atomic-updates
      source[key] = val;
      for (const { resolve } of listener) {
        resolve(val);
      }
      return val as any;
    } catch (e) {
      (source as any)[errKey] = e;
      for (const { reject } of listener) {
        reject(e);
      }
      throw e;
    } finally {
      listeners.delete(source);
    }
  };
};
