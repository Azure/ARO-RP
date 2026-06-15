import { generateHash } from "@scalar/helpers/string/generate-hash";
function getHash(value) {
  const hashHex = generateHash(value);
  const hash = hashHex.substring(0, 7);
  return hash.match(/^\d+$/) ? "a" + hash.substring(1) : hash;
}
async function generateUniqueValue(compress, value, compressedToValue, prevCompressedValue, depth = 0) {
  const MAX_DEPTH = 100;
  if (depth >= MAX_DEPTH) {
    throw "Can not generate unique compressed values";
  }
  const compressedValue = await compress(prevCompressedValue ?? value);
  if (compressedToValue[compressedValue] !== void 0 && compressedToValue[compressedValue] !== value) {
    return generateUniqueValue(compress, value, compressedToValue, compressedValue, depth + 1);
  }
  compressedToValue[compressedValue] = value;
  return compressedValue;
}
const uniqueValueGeneratorFactory = (compress, compressedToValue) => {
  const valueToCompressed = Object.fromEntries(Object.entries(compressedToValue).map(([key, value]) => [value, key]));
  return {
    /**
     * Generates a unique compressed value for the given input string.
     * First checks if a compressed value already exists in the cache.
     * If not, generates a new unique compressed value and stores it in the cache.
     *
     * @param value - The original string value to compress
     * @returns A Promise that resolves to the compressed string value
     *
     * @example
     * const generator = uniqueValueGeneratorFactory(compress, {})
     * const compressed = await generator.generate('example.com/schema.json')
     * // Returns a unique compressed value like 'example'
     */
    generate: async (value) => {
      const cache = valueToCompressed[value];
      if (cache) {
        return cache;
      }
      const generatedValue = await generateUniqueValue(compress, value, compressedToValue);
      const compressedValue = generatedValue.match(/^\d+$/) ? `a${generatedValue}` : generatedValue;
      valueToCompressed[value] = compressedValue;
      return compressedValue;
    }
  };
};
export {
  generateUniqueValue,
  getHash,
  uniqueValueGeneratorFactory
};
//# sourceMappingURL=value-generator.js.map
