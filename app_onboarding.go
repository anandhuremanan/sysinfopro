package main

import (
	"log/slog"
	"sync"

	"sysinfopro/internal/appdata"
	"sysinfopro/internal/settings"
)

// What the welcome screens and the guided tour have done so far, kept with the
// other settings so a reset embedded browser does not bring them back
type onboardingState struct {
	WelcomeDone bool `json:"welcome_done"`
	TourOffered bool `json:"tour_offered"`
}

const onboardingFile = "onboarding.json"

// Onboarding tells the screens what to show a new user
type Onboarding struct {
	// the welcome screens, on the first start after installing
	ShowWelcome bool `json:"show_welcome"`
	// the "take a tour" card on the home screen, once the welcome is done
	OfferTour bool `json:"offer_tour"`
}

var onboardingMu sync.Mutex

func loadOnboarding() onboardingState {
	state, found, err := settings.LoadExisting(onboardingFile, onboardingState{})
	if err != nil {
		slog.Warn("onboarding settings unreadable", "err", err)
		return onboardingState{WelcomeDone: true, TourOffered: true}
	}
	if !found {
		if appdata.FreshInstall() {
			// saved straight away, so closing the app midway through the welcome
			// still shows it next time
			state = onboardingState{}
		} else {
			// someone updating from a version without the welcome: they know the app
			state = onboardingState{WelcomeDone: true, TourOffered: true}
		}
		saveOnboarding(state)
	}
	return state
}

func saveOnboarding(state onboardingState) {
	if err := settings.Save(onboardingFile, state); err != nil {
		slog.Warn("could not save onboarding settings", "err", err)
	}
}

// GetOnboarding reports whether to show the welcome screens or offer the tour
func (a *App) GetOnboarding() Onboarding {
	onboardingMu.Lock()
	defer onboardingMu.Unlock()
	state := loadOnboarding()
	return Onboarding{ShowWelcome: !state.WelcomeDone, OfferTour: state.WelcomeDone && !state.TourOffered}
}

// FinishWelcome records that the welcome screens were seen or skipped
func (a *App) FinishWelcome() {
	onboardingMu.Lock()
	defer onboardingMu.Unlock()
	state := loadOnboarding()
	state.WelcomeDone = true
	saveOnboarding(state)
}

// FinishTourOffer records that the tour was taken or turned down, so the home
// screen stops offering it; Settings can still start it
func (a *App) FinishTourOffer() {
	onboardingMu.Lock()
	defer onboardingMu.Unlock()
	state := loadOnboarding()
	state.TourOffered = true
	saveOnboarding(state)
}
