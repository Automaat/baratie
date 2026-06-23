import {
	LayoutDashboard,
	BookOpen,
	Apple,
	Refrigerator,
	CalendarDays,
	Settings
} from 'lucide-svelte';

export const NAV_ROUTES = [
	{ href: '/', label: 'Pulpit', icon: LayoutDashboard },
	{ href: '/recipes', label: 'Przepisy', icon: BookOpen },
	{ href: '/foods', label: 'Baza produktów', icon: Apple },
	{ href: '/pantry', label: 'Spiżarnia', icon: Refrigerator },
	{ href: '/meal-plan', label: 'Plan posiłków', icon: CalendarDays },
	{ href: '/settings', label: 'Ustawienia', icon: Settings }
] as const;

export type NavRoute = (typeof NAV_ROUTES)[number];

export const NAV_HREFS: ReadonlySet<string> = new Set(NAV_ROUTES.map((r) => r.href));
