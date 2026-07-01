import { parseCookie, stringifySetCookie } from '../../../node_modules/cookie/dist/index.js';

export const parse = parseCookie;

/**
 * @param {string} name
 * @param {string} value
 * @param {{ encode?: (value: string) => string, [key: string]: unknown }} [options]
 */
export function serialize(name, value, options = {}) {
	const { encode, ...cookie } = options;

	return stringifySetCookie({ name, value, ...cookie }, encode ? { encode } : undefined);
}
