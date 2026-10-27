// A short message that clears itself after a while. Setting a new message restarts
// the timer, so an older timer never wipes a newer message early.
export class Status {
	#text = $state('');
	#timer: ReturnType<typeof setTimeout> | undefined;
	#ms: number;

	constructor(ms = 5000) {
		this.#ms = ms;
	}

	get text(): string {
		return this.#text;
	}

	set text(value: string | undefined) {
		this.show(value ?? '');
	}

	// show a message; ms overrides how long it stays (0 = until replaced)
	show(text: string, ms = this.#ms) {
		clearTimeout(this.#timer);
		this.#text = text;
		if (text && ms > 0) this.#timer = setTimeout(() => (this.#text = ''), ms);
	}

	clear() {
		clearTimeout(this.#timer);
		this.#text = '';
	}
}
