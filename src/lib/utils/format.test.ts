import { describe, it, expect } from 'vitest';
import { formatDate, formatQuantity, formatMinutes, mealTypeLabel } from './format';

describe('formatDate', () => {
	it('formats a YYYY-MM-DD string as DD.MM.YYYY', () => {
		expect(formatDate('2026-03-15')).toBe('15.03.2026');
	});

	it('returns an em-dash for empty or invalid input', () => {
		expect(formatDate(null)).toBe('—');
		expect(formatDate(undefined)).toBe('—');
		expect(formatDate('not-a-date')).toBe('—');
	});
});

describe('formatQuantity', () => {
	it('drops trailing zeros and appends the unit', () => {
		expect(formatQuantity(2, 'kg')).toBe('2 kg');
		expect(formatQuantity(1.5)).toBe('1,5');
	});
});

describe('formatMinutes', () => {
	it('renders hours and minutes', () => {
		expect(formatMinutes(0)).toBe('—');
		expect(formatMinutes(45)).toBe('45 min');
		expect(formatMinutes(60)).toBe('1 h');
		expect(formatMinutes(80)).toBe('1 h 20 min');
	});
});

describe('mealTypeLabel', () => {
	it('maps known codes and passes through unknown ones', () => {
		expect(mealTypeLabel('breakfast')).toBe('Śniadanie');
		expect(mealTypeLabel('mystery')).toBe('mystery');
	});
});
