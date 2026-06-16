const array = [];
const characterCodeCache = [];

export default function leven(first, second, options) {
	if (first === second) {
		return 0;
	}

	const maxDistance = options?.maxDistance;
	const swap = first;

	// Swapping the strings if `a` is longer than `b` so we know which one is the
	// shortest & which one is the longest
	if (first.length > second.length) {
		first = second;
		second = swap;
	}

	let firstLength = first.length;
	let secondLength = second.length;

	// Performing suffix trimming:
	// We can linearly drop suffix common to both strings since they
	// don't increase distance at all
	// Note: `~-` is the bitwise way to perform a `- 1` operation
	while (firstLength > 0 && (first.charCodeAt(~-firstLength) === second.charCodeAt(~-secondLength))) {
		firstLength--;
		secondLength--;
	}

	// Performing prefix trimming
	// We can linearly drop prefix common to both strings since they
	// don't increase distance at all
	let start = 0;

	while (start < firstLength && (first.charCodeAt(start) === second.charCodeAt(start))) {
		start++;
	}

	firstLength -= start;
	secondLength -= start;

	// Early termination after trimming: if difference in length exceeds max distance
	if (maxDistance !== undefined && secondLength - firstLength > maxDistance) {
		return maxDistance;
	}

	if (firstLength === 0) {
		return maxDistance !== undefined && secondLength > maxDistance
			? maxDistance
			: secondLength;
	}

	let bCharacterCode;
	let result;
	let temporary;
	let temporary2;
	let index = 0;
	let index2 = 0;

	while (index < firstLength) {
		characterCodeCache[index] = first.charCodeAt(start + index);
		array[index] = ++index;
	}

	while (index2 < secondLength) {
		bCharacterCode = second.charCodeAt(start + index2);
		temporary = index2++;
		result = index2;

		for (index = 0; index < firstLength; index++) {
			temporary2 = bCharacterCode === characterCodeCache[index] ? temporary : temporary + 1;
			temporary = array[index];
			// eslint-disable-next-line no-multi-assign
			result = array[index] = temporary > result
				? (temporary2 > result ? result + 1 : temporary2)
				: (temporary2 > temporary ? temporary + 1 : temporary2);
		}

		// Early termination: if all values in current row exceed maxDistance
		if (maxDistance !== undefined) {
			let rowMinimum = result;
			for (index = 0; index < firstLength; index++) {
				if (array[index] < rowMinimum) {
					rowMinimum = array[index];
				}
			}

			if (rowMinimum > maxDistance) {
				return maxDistance;
			}
		}
	}

	// Bound arrays to avoid retaining large previous sizes
	array.length = firstLength;
	characterCodeCache.length = firstLength;

	return maxDistance !== undefined && result > maxDistance ? maxDistance : result;
}

export function closestMatch(target, candidates, options) {
	if (!Array.isArray(candidates) || candidates.length === 0) {
		return undefined;
	}

	const userMax = options?.maxDistance;
	const targetLength = target.length;

	// Exact match fast-path
	for (const candidate of candidates) {
		if (candidate === target) {
			return candidate;
		}
	}

	if (userMax === 0) {
		return undefined;
	}

	let best;
	let bestDist = Number.POSITIVE_INFINITY;
	const seen = new Set();

	for (const candidate of candidates) {
		if (seen.has(candidate)) {
			continue;
		}

		seen.add(candidate);

		const lengthDiff = Math.abs(candidate.length - targetLength);
		if (lengthDiff >= bestDist) {
			continue;
		}

		if (userMax !== undefined && lengthDiff > userMax) {
			continue;
		}

		const cap = Number.isFinite(bestDist)
			? (userMax === undefined ? bestDist : Math.min(bestDist, userMax))
			: userMax;

		const distance = cap === undefined
			? leven(target, candidate)
			: leven(target, candidate, {maxDistance: cap});

		// Skip candidates that exceed the user's maximum distance
		if (userMax !== undefined && distance > userMax) {
			continue;
		}

		// If we got a capped result that equals the cap, we need the actual distance
		// for accurate comparison, but only if the cap was due to userMax
		let actualD = distance;
		if (cap !== undefined && distance === cap && cap === userMax) {
			actualD = leven(target, candidate);
		}

		if (actualD < bestDist) {
			bestDist = actualD;
			best = candidate;
			if (bestDist === 0) {
				break;
			}
		}
	}

	if (userMax !== undefined && bestDist > userMax) {
		return undefined;
	}

	return best;
}
