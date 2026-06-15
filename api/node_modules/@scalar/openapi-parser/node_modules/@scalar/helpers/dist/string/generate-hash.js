const generateHash = (input) => {
  const seed = 0;
  let h1 = seed;
  let h2 = seed;
  const len = input.length;
  const remainder = len & 15;
  const bytes = len - remainder;
  const c1 = 2277735313;
  const c2 = 1291169091;
  const c3 = 1390208809;
  const c4 = 944331445;
  for (let i = 0; i < bytes; i += 16) {
    let k1 = input.charCodeAt(i) & 255 | (input.charCodeAt(i + 1) & 255) << 8 | (input.charCodeAt(i + 2) & 255) << 16 | (input.charCodeAt(i + 3) & 255) << 24;
    let k2 = input.charCodeAt(i + 4) & 255 | (input.charCodeAt(i + 5) & 255) << 8 | (input.charCodeAt(i + 6) & 255) << 16 | (input.charCodeAt(i + 7) & 255) << 24;
    let k3 = input.charCodeAt(i + 8) & 255 | (input.charCodeAt(i + 9) & 255) << 8 | (input.charCodeAt(i + 10) & 255) << 16 | (input.charCodeAt(i + 11) & 255) << 24;
    let k4 = input.charCodeAt(i + 12) & 255 | (input.charCodeAt(i + 13) & 255) << 8 | (input.charCodeAt(i + 14) & 255) << 16 | (input.charCodeAt(i + 15) & 255) << 24;
    k1 = Math.imul(k1, c1);
    k1 = k1 << 15 | k1 >>> 17;
    k1 = Math.imul(k1, c2);
    h1 ^= k1;
    h1 = h1 << 13 | h1 >>> 19;
    h1 = Math.imul(h1, 5) + 3864292196;
    k2 = Math.imul(k2, c2);
    k2 = k2 << 16 | k2 >>> 16;
    k2 = Math.imul(k2, c3);
    h2 ^= k2;
    h2 = h2 << 17 | h2 >>> 15;
    h2 = Math.imul(h2, 5) + 461845907;
    k3 = Math.imul(k3, c3);
    k3 = k3 << 17 | k3 >>> 15;
    k3 = Math.imul(k3, c4);
    h1 ^= k3;
    h1 = h1 << 15 | h1 >>> 17;
    h1 = Math.imul(h1, 5) + 1390208809;
    k4 = Math.imul(k4, c4);
    k4 = k4 << 18 | k4 >>> 14;
    k4 = Math.imul(k4, c1);
    h2 ^= k4;
    h2 = h2 << 13 | h2 >>> 19;
    h2 = Math.imul(h2, 5) + 944331445;
  }
  if (remainder > 0) {
    let k1 = 0;
    let k2 = 0;
    let k3 = 0;
    let k4 = 0;
    if (remainder >= 15) {
      k4 ^= (input.charCodeAt(bytes + 14) & 255) << 16;
    }
    if (remainder >= 14) {
      k4 ^= (input.charCodeAt(bytes + 13) & 255) << 8;
    }
    if (remainder >= 13) {
      k4 ^= input.charCodeAt(bytes + 12) & 255;
      k4 = Math.imul(k4, c4);
      k4 = k4 << 18 | k4 >>> 14;
      k4 = Math.imul(k4, c1);
      h2 ^= k4;
    }
    if (remainder >= 12) {
      k3 ^= (input.charCodeAt(bytes + 11) & 255) << 24;
    }
    if (remainder >= 11) {
      k3 ^= (input.charCodeAt(bytes + 10) & 255) << 16;
    }
    if (remainder >= 10) {
      k3 ^= (input.charCodeAt(bytes + 9) & 255) << 8;
    }
    if (remainder >= 9) {
      k3 ^= input.charCodeAt(bytes + 8) & 255;
      k3 = Math.imul(k3, c3);
      k3 = k3 << 17 | k3 >>> 15;
      k3 = Math.imul(k3, c4);
      h1 ^= k3;
    }
    if (remainder >= 8) {
      k2 ^= (input.charCodeAt(bytes + 7) & 255) << 24;
    }
    if (remainder >= 7) {
      k2 ^= (input.charCodeAt(bytes + 6) & 255) << 16;
    }
    if (remainder >= 6) {
      k2 ^= (input.charCodeAt(bytes + 5) & 255) << 8;
    }
    if (remainder >= 5) {
      k2 ^= input.charCodeAt(bytes + 4) & 255;
      k2 = Math.imul(k2, c2);
      k2 = k2 << 16 | k2 >>> 16;
      k2 = Math.imul(k2, c3);
      h2 ^= k2;
    }
    if (remainder >= 4) {
      k1 ^= (input.charCodeAt(bytes + 3) & 255) << 24;
    }
    if (remainder >= 3) {
      k1 ^= (input.charCodeAt(bytes + 2) & 255) << 16;
    }
    if (remainder >= 2) {
      k1 ^= (input.charCodeAt(bytes + 1) & 255) << 8;
    }
    if (remainder >= 1) {
      k1 ^= input.charCodeAt(bytes) & 255;
      k1 = Math.imul(k1, c1);
      k1 = k1 << 15 | k1 >>> 17;
      k1 = Math.imul(k1, c2);
      h1 ^= k1;
    }
  }
  h1 ^= len;
  h2 ^= len;
  h1 += h2;
  h2 += h1;
  h1 ^= h1 >>> 16;
  h1 = Math.imul(h1, 2246822507);
  h1 ^= h1 >>> 13;
  h1 = Math.imul(h1, 3266489909);
  h1 ^= h1 >>> 16;
  h2 ^= h2 >>> 16;
  h2 = Math.imul(h2, 2246822507);
  h2 ^= h2 >>> 13;
  h2 = Math.imul(h2, 3266489909);
  h2 ^= h2 >>> 16;
  h1 += h2;
  h2 += h1;
  return (h1 >>> 0).toString(16).padStart(8, "0") + (h2 >>> 0).toString(16).padStart(8, "0");
};
export {
  generateHash
};
//# sourceMappingURL=generate-hash.js.map
