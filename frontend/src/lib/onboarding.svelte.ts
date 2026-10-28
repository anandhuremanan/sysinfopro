import { FinishTourOffer, FinishWelcome, GetOnboarding } from '$lib/wailsjs/go/main/App';

// The welcome screens (first start after installing) and the guided tour of
// the home screen. Both can be started again from Settings.
export const onboarding = $state({
	welcome: false,
	// the "take a tour" card on the home screen
	invite: false,
	tour: false
});

export async function loadOnboarding() {
	try {
		const state = await GetOnboarding();
		onboarding.welcome = state.show_welcome;
		onboarding.invite = state.offer_tour;
	} catch (err) {
		console.error('Failed to read onboarding state:', err);
	}
}

// from Settings
export function showWelcome() {
	onboarding.tour = false;
	onboarding.welcome = true;
}

// seen or skipped; a first-time user is then offered the tour
export async function finishWelcome() {
	onboarding.welcome = false;
	try {
		await FinishWelcome();
		onboarding.invite = (await GetOnboarding()).offer_tour;
	} catch (err) {
		console.error('Failed to save onboarding state:', err);
	}
}

export function startTour() {
	dismissInvite();
	onboarding.welcome = false;
	onboarding.tour = true;
}

export function endTour() {
	onboarding.tour = false;
}

// taken or turned down: the home screen stops offering it
export function dismissInvite() {
	if (!onboarding.invite) return;
	onboarding.invite = false;
	FinishTourOffer().catch((err) => console.error('Failed to save onboarding state:', err));
}
