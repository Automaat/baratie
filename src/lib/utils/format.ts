// Shared display formatters. All UI is Polish, so dates and numbers use the
// pl-PL locale.

const dateFormatter = new Intl.DateTimeFormat('pl-PL', {
	year: 'numeric',
	month: '2-digit',
	day: '2-digit'
});

// formatDate renders a YYYY-MM-DD (or ISO) string as DD.MM.YYYY. Invalid or
// empty input renders an em-dash so the UI never shows "Invalid Date".
export function formatDate(value: string | null | undefined): string {
	if (!value) return '—';
	const date = new Date(value.length === 10 ? `${value}T00:00:00` : value);
	if (Number.isNaN(date.getTime())) return '—';
	return dateFormatter.format(date);
}

// formatQuantity renders a number with up to two decimals and an optional unit,
// dropping trailing zeros (2.00 → "2", 1.50 → "1,5").
export function formatQuantity(value: number, unit?: string): string {
	const num = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 2 }).format(value);
	return unit ? `${num} ${unit}` : num;
}

// formatMinutes renders a minute count as "1 h 20 min" / "45 min" / "—".
export function formatMinutes(total: number): string {
	if (!total || total <= 0) return '—';
	const hours = Math.floor(total / 60);
	const minutes = total % 60;
	if (hours === 0) return `${minutes} min`;
	if (minutes === 0) return `${hours} h`;
	return `${hours} h ${minutes} min`;
}

// MEAL_TYPE_LABELS maps backend meal_type codes to Polish labels.
export const MEAL_TYPE_LABELS: Record<string, string> = {
	breakfast: 'Śniadanie',
	lunch: 'Obiad',
	dinner: 'Kolacja',
	snack: 'Przekąska'
};

export function mealTypeLabel(code: string): string {
	return MEAL_TYPE_LABELS[code] ?? code;
}
