// The way back through the app's screens, kept apart from the screen list so it
// can be tested without loading every screen.

// The way back: a stack of the screens visited. Opening one already in the stack
// returns to it instead of piling up loops; Home starts over.
export function pushView<V extends string>(stack: V[], target: V): V[] {
	if (target === 'home') return [target];
	const at = stack.indexOf(target);
	if (at >= 0) return stack.slice(0, at + 1);
	return [...stack, target];
}

export function popView<V extends string>(stack: V[]): V[] {
	return stack.length > 1 ? stack.slice(0, -1) : stack;
}
