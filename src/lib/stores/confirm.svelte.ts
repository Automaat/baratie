// Programmatic confirm dialog: await confirm(...) anywhere in browser code,
// get back a boolean. A single <Confirm /> instance mounted in the root layout
// renders the modal driven by this store.

export interface ConfirmOptions {
	title: string;
	message: string;
	confirmText?: string;
	cancelText?: string;
	danger?: boolean;
	onConfirm?: () => void | Promise<void>;
}

export interface ConfirmRequest extends ConfirmOptions {
	pending: boolean;
	resolve: (ok: boolean) => void;
}

let current = $state<ConfirmRequest | null>(null);

function open(options: ConfirmOptions): Promise<boolean> {
	if (current) {
		if (current.pending) {
			// The first request's handler is mid-flight — clobbering it would
			// orphan its Promise. Reject the newcomer instead.
			return Promise.resolve(false);
		}
		// A second confirm before the first resolved cancels the first.
		current.resolve(false);
		current = null;
	}
	return new Promise<boolean>((resolve) => {
		current = { ...options, pending: false, resolve };
	});
}

async function confirmAction(): Promise<void> {
	if (!current || current.pending) return;
	const req = current;
	if (!req.onConfirm) {
		current = null;
		req.resolve(true);
		return;
	}
	req.pending = true;
	try {
		await req.onConfirm();
		current = null;
		req.resolve(true);
	} catch {
		current = null;
		req.resolve(false);
	}
}

function cancelAction(): void {
	if (!current || current.pending) return;
	const req = current;
	current = null;
	req.resolve(false);
}

export const confirmDialog = {
	get current(): ConfirmRequest | null {
		return current;
	},
	confirm: confirmAction,
	cancel: cancelAction
};

export function confirm(options: ConfirmOptions): Promise<boolean> {
	return open(options);
}
