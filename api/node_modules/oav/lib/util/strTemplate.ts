export type TemplateFunc<R> = (dict: any) => R;

export function strTemplate<T extends Array<string | number>>(
  strings: readonly string[],
  ...keys: T
): TemplateFunc<string> {
  return ((...values: any) => {
    const dict = values[values.length - 1] || {};
    const result = [strings[0]];
    keys.forEach((key, i) => {
      const value = Number.isInteger(key as number) ? values[key] : dict[key];
      result.push(value, strings[i + 1]);
    });
    return result.join("");
  }) as any;
}
