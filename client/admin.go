//go:build js && wasm

// The admin console, as a GoWebComponents/WASM app. It is pure client-side UI that talks to the Go
// backend only over gRPC (AdminService, via the GoGRPCBridge WebSocket tunnel) — no server-rendered
// HTML, no HTTP forms. The JWT from Login is held in localStorage and sent in call metadata.
package main

import (
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// cashfluxUsersPageSize is the page size for the CashFlux user list — small enough to be a quick
// call, large enough that most deployments (an admin-invited handful of accounts) never need
// "load more" at all.
const cashfluxUsersPageSize = 50

// AdminApp is the root admin component: a login gate, then the console (anime / résumé / settings).
func AdminApp() ui.Node {
	token := ui.UseState(loadToken())
	view := ui.UseState(currentAdminView())
	flash := ui.UseState("")

	username := ui.UseState("")
	password := ui.UseState("")

	query := ui.UseState("")
	results := ui.UseState[[]*sitepb.Anime](nil)
	tracked := ui.UseState[[]*sitepb.Anime](nil)

	jobURL := ui.UseState("")
	canonical := ui.UseState[*sitepb.Resume](nil) // the PERMANENT base résumé (diff baseline)
	tailored := ui.UseState[*sitepb.Resume](nil)  // the proposed tailored résumé (nil = none pending)
	jobAnalysis := ui.UseState[*sitepb.JobAnalysis](nil)
	rationales := ui.UseState[[]*sitepb.Rationale](nil)
	variants := ui.UseState[[]*sitepb.TailoringMeta](nil) // saved tailoring variants (the CRUD list)

	// RSS / anime control panel
	promptText := ui.UseState("")                   // the single QOTD generation instruction
	dryRun := ui.UseState[*sitepb.PostPreview](nil) // last dry-run preview
	dryRunning := ui.UseState(false)                // dry-run in flight
	slackWebhook := ui.UseState("")
	slackSet := ui.UseState(false)
	slackEnabled := ui.UseState(false)
	slackHour := ui.UseState(9) // daily scheduled-post hour (0–23)

	keySet := ui.UseState(false)
	model := ui.UseState("")
	models := ui.UseState[[]string](nil)
	apiKey := ui.UseState("")

	// CashFlux device pairing
	cashfluxConfigured := ui.UseState(true) // flips false only on a FailedPrecondition from the server
	cashfluxPending := ui.UseState[[]*sitepb.CashFluxPendingDevice](nil)
	cashfluxBusy := ui.UseState[map[string]bool](nil)                     // device_ids currently being approved/rejected — a set, so one row's in-flight request doesn't re-enable another row's buttons
	cashfluxJustPaired := ui.UseState[*sitepb.CashFluxPendingDevice](nil) // the device that was just approved
	cashfluxPairingCode := ui.UseState("")                                // the pairing code from that approval, shown once until the tab reloads
	cashfluxCopied := ui.UseState(false)                                  // flips true after a successful copy; reset on the next approval

	// CashFlux activation codes — the primary way a device gets in. Held until the tab reloads or
	// another code is minted; each one is single-use and short-lived, so a stale one on screen is
	// harmless, not a secret worth clearing eagerly.
	cashfluxActivationCode := ui.UseState("")
	cashfluxActivationExpires := ui.UseState("") // RFC3339 from the server; "" when nothing is minted
	cashfluxActivationMinting := ui.UseState(false)
	cashfluxActivationCopied := ui.UseState(false)

	// CashFlux users + storage (same tab, alongside pending-device pairing above)
	cashfluxUsers := ui.UseState[[]*sitepb.CashFluxUser](nil)
	cashfluxUsersMore := ui.UseState(false) // true when the last page came back full — more may exist
	cashfluxUsersLoading := ui.UseState(false)
	cashfluxStorage := ui.UseState[*sitepb.CashFluxStorageStats](nil)

	// Auth flow: first-run setup + password reset (the owner has no session in these cases).
	needsSetup := ui.UseState(false)
	recoveryHint := ui.UseState("")
	authScreen := ui.UseState("login") // "login" | "reset"
	setupHint := ui.UseState("")
	setupToken := ui.UseState("")
	newPass := ui.UseState("")
	phraseInput := ui.UseState("")
	recoveryPhrase := ui.UseState("") // a freshly generated phrase, shown once
	phraseIsSetup := ui.UseState(false)
	pendingToken := ui.UseState("") // session token held until the recovery phrase is acknowledged

	authed := token.Get() != ""

	// onAuthErr clears the session when a call is rejected as unauthenticated (expired/forged token).
	onAuthErr := func(err error) bool {
		if status.Code(err) == codes.Unauthenticated {
			clearToken()
			token.Set("")
			flash.Set("session expired — sign in again")
			return true
		}
		return false
	}

	// Load the active view's data when signed in or the view changes.
	ui.UseEffect(func() func() {
		if !authed {
			return nil
		}
		switch view.Get() {
		case "anime":
			go func() {
				c, err := adminClient()
				if err != nil {
					flash.Set("connection error")
					return
				}
				ctx, cancel := callCtx(token.Get())
				defer cancel()
				list, err := c.ListTracked(ctx, &sitepb.Empty{})
				if onAuthErr(err) || err != nil {
					return
				}
				tracked.Set(list.GetItems())
			}()
		case "resume":
			if canonical.Get() == nil { // load once: the permanent base résumé + the saved variants list
				go func() {
					c, err := adminClient()
					if err != nil {
						flash.Set("connection error")
						return
					}
					ctx, cancel := callCtx(token.Get())
					defer cancel()
					r, err := c.GetBaseResume(ctx, &sitepb.Empty{})
					if onAuthErr(err) {
						return
					}
					if err == nil {
						canonical.Set(r)
					}
					if l, err := c.ListTailorings(ctx, &sitepb.Empty{}); err == nil {
						variants.Set(l.GetItems())
					}
				}()
			}
		case "settings":
			go func() {
				c, err := adminClient()
				if err != nil {
					flash.Set("connection error")
					return
				}
				ctx, cancel := callCtx(token.Get())
				defer cancel()
				s, err := c.GetSettings(ctx, &sitepb.Empty{})
				if onAuthErr(err) || err != nil {
					return
				}
				keySet.Set(s.GetKeySet())
				model.Set(s.GetOpenaiModel())
				ml, err := c.ListModels(ctx, &sitepb.Empty{})
				if err == nil {
					models.Set(ml.GetModels())
				}
			}()
		case "rss":
			go func() {
				c, err := adminClient()
				if err != nil {
					flash.Set("connection error")
					return
				}
				ctx, cancel := callCtx(token.Get())
				defer cancel()
				gp, err := c.GetPrompt(ctx, &sitepb.Empty{})
				if onAuthErr(err) {
					return
				}
				// Only seed the textarea from the server when it's still empty — never clobber edits the
				// owner has already typed (this effect re-fires on every nav back to the RSS view, and a
				// slow first-load response could otherwise land after they've started editing).
				if err == nil && promptText.Get() == "" {
					promptText.Set(gp.GetText())
				}
				if sc, err := c.GetSlackConfig(ctx, &sitepb.Empty{}); err == nil {
					slackSet.Set(sc.GetWebhookSet())
					slackEnabled.Set(sc.GetEnabled())
					slackHour.Set(int(sc.GetPostHour()))
				}
			}()
		case "cashflux":
			// Clear any pairing-code callout left over from a previous visit to this tab — otherwise
			// navigating away and back would show a stale "read this to <device>" prompt for a device
			// that's no longer pending, which looks like a live cross-check but isn't.
			cashfluxJustPaired.Set(nil)
			cashfluxPairingCode.Set("")
			cashfluxCopied.Set(false)
			// Same reasoning for the activation code: a code minted on a previous visit has almost
			// certainly expired (5-minute TTL), so showing it again would invite typing a dead code.
			cashfluxActivationCode.Set("")
			cashfluxActivationExpires.Set("")
			cashfluxActivationCopied.Set(false)
			go func() {
				c, err := adminClient()
				if err != nil {
					flash.Set("connection error")
					return
				}
				ctx, cancel := callCtx(token.Get())
				defer cancel()
				pending, err := c.ListCashFluxPendingDevices(ctx, &sitepb.Empty{})
				switch {
				case onAuthErr(err):
					return
				case status.Code(err) == codes.FailedPrecondition:
					cashfluxConfigured.Set(false)
					return
				case err != nil:
					flash.Set("couldn't load pending devices: " + err.Error())
					return
				default:
					cashfluxConfigured.Set(true)
					cashfluxPending.Set(pending.GetItems())
				}
				// Users + storage stats load alongside pending devices — same tab, same gate.
				cashfluxUsersLoading.Set(true)
				users, err := c.ListCashFluxUsers(ctx, &sitepb.CashFluxListUsersRequest{Limit: cashfluxUsersPageSize})
				cashfluxUsersLoading.Set(false)
				switch {
				case onAuthErr(err):
					return
				case err != nil:
					flash.Set("couldn't load users: " + err.Error())
				default:
					cashfluxUsers.Set(users.GetItems())
					cashfluxUsersMore.Set(len(users.GetItems()) == cashfluxUsersPageSize)
				}
				stats, err := c.GetCashFluxStorageStats(ctx, &sitepb.Empty{})
				switch {
				case onAuthErr(err):
					return
				case err != nil:
					flash.Set("couldn't load storage stats: " + err.Error())
				default:
					cashfluxStorage.Set(stats)
				}
			}()
		}
		return nil
	}, authed, view.Get())

	// Sync the view when the user navigates back/forward in the browser.
	ui.UseEffect(func() func() {
		return onPopState(func() { view.Set(currentAdminView()) })
	}, "popstate-mount")

	// Discover whether the deployed site still needs first-run setup, and the reset hint. Runs once.
	ui.UseEffect(func() func() {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx("")
			defer cancel()
			if st, err := c.AuthState(ctx, &sitepb.Empty{}); err == nil {
				needsSetup.Set(st.GetNeedsSetup())
				recoveryHint.Set(st.GetRecoveryHint())
			}
		}()
		return func() {}
	}, "authstate-mount")

	if !authed {
		// A freshly generated recovery phrase is shown once before proceeding. Handlers here use
		// WrapHandler (not the hook-indexed UseEvent) so the auth sub-screens can branch freely.
		if recoveryPhrase.Get() != "" {
			onContinue := ui.WrapHandler(func() {
				isSetup := phraseIsSetup.Get()
				recoveryPhrase.Set("")
				if isSetup {
					tok := pendingToken.Get()
					pendingToken.Set("")
					saveToken(tok)
					token.Set(tok)
				} else {
					authScreen.Set("login")
				}
			})
			return phraseView(recoveryPhrase.Get(), phraseIsSetup.Get(), onContinue)
		}

		if needsSetup.Get() {
			onSetup := ui.WrapHandler(func() {
				flash.Set("")
				go func() {
					c, err := adminClient()
					if err != nil {
						flash.Set("connection error: " + err.Error())
						return
					}
					ctx, cancel := callCtx("")
					defer cancel()
					rep, err := c.Setup(ctx, &sitepb.SetupRequest{
						Username: username.Get(), Password: password.Get(),
						Hint: setupHint.Get(), SetupToken: setupToken.Get(),
					})
					if err != nil {
						flash.Set("setup failed: " + err.Error())
						return
					}
					if !rep.GetOk() {
						flash.Set(rep.GetError())
						return
					}
					needsSetup.Set(false)
					pendingToken.Set(rep.GetToken())
					phraseIsSetup.Set(true)
					recoveryPhrase.Set(rep.GetRecoveryPhrase())
				}()
			})
			return setupView(username, password, setupHint, setupToken, onSetup, flash.Get())
		}

		if authScreen.Get() == "reset" {
			onReset := ui.WrapHandler(func() {
				flash.Set("")
				go func() {
					c, err := adminClient()
					if err != nil {
						flash.Set("connection error: " + err.Error())
						return
					}
					ctx, cancel := callCtx("")
					defer cancel()
					rep, err := c.ResetPassword(ctx, &sitepb.ResetRequest{
						RecoveryPhrase: phraseInput.Get(), NewPassword: newPass.Get(),
					})
					if err != nil {
						flash.Set("reset failed: " + err.Error())
						return
					}
					if !rep.GetOk() {
						flash.Set(rep.GetError())
						return
					}
					phraseInput.Set("")
					newPass.Set("")
					phraseIsSetup.Set(false)
					recoveryPhrase.Set(rep.GetRecoveryPhrase())
				}()
			})
			onBack := ui.WrapHandler(func() { flash.Set(""); authScreen.Set("login") })
			return resetView(recoveryHint.Get(), phraseInput, newPass, onReset, onBack, flash.Get())
		}

		onLogin := ui.WrapHandler(func() {
			flash.Set("")
			go func() {
				c, err := adminClient()
				if err != nil {
					flash.Set("connection error: " + err.Error())
					return
				}
				ctx, cancel := callCtx("")
				defer cancel()
				rep, err := c.Login(ctx, &sitepb.LoginRequest{Username: username.Get(), Password: password.Get()})
				if err != nil {
					flash.Set("login failed: " + err.Error())
					return
				}
				if !rep.GetOk() {
					flash.Set("wrong username or password")
					return
				}
				saveToken(rep.GetToken())
				token.Set(rep.GetToken())
			}()
		})
		onForgot := ui.WrapHandler(func() { flash.Set(""); authScreen.Set("reset") })
		return loginView(username, password, onLogin, onForgot, flash.Get())
	}

	// Console handlers.
	onLogout := ui.UseEvent(func() { clearToken(); token.Set(""); results.Set(nil); tracked.Set(nil) })
	navTo := func(v string) ui.Handler {
		return ui.WrapHandler(func() { flash.Set(""); pushAdminPath(v); view.Set(v) })
	}

	onSearch := ui.UseEvent(func() {
		flash.Set("")
		go func() {
			c, err := adminClient()
			if err != nil {
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			res, err := c.SearchAnime(ctx, &sitepb.SearchRequest{Query: query.Get()})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("search failed")
				return
			}
			results.Set(res.GetItems())
		}()
	})

	reload := func() {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if list, err := c.ListTracked(ctx, &sitepb.Empty{}); err == nil {
				tracked.Set(list.GetItems())
			}
			if query.Get() != "" {
				if res, err := c.SearchAnime(ctx, &sitepb.SearchRequest{Query: query.Get()}); err == nil {
					results.Set(res.GetItems())
				}
			}
		}()
	}
	trackFn := func(id int32, endpointTrack bool) {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if endpointTrack {
				_, err = c.TrackAnime(ctx, &sitepb.AnimeId{AnilistId: id})
			} else {
				_, err = c.UntrackAnime(ctx, &sitepb.AnimeId{AnilistId: id})
			}
			if !onAuthErr(err) {
				reload()
			}
		}()
	}
	onCheck := ui.UseEvent(func() {
		flash.Set("running release check…")
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			rep, err := c.RunReleaseCheck(ctx, &sitepb.Empty{})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("release check failed")
				return
			}
			flash.Set("release check done")
			_ = rep
			reload()
		}()
	})

	reloadVariants := func() {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if l, err := c.ListTailorings(ctx, &sitepb.Empty{}); err == nil {
				variants.Set(l.GetItems())
			}
		}()
	}

	onTailor := ui.UseEvent(func() {
		flash.Set("tailoring…")
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			r, err := c.TailorResume(ctx, &sitepb.TailorRequest{JobUrl: jobURL.Get()})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("tailoring failed: " + status.Convert(err).Message())
				return
			}
			flash.Set("tailored — review the diff, extraction, and rationale below")
			tailored.Set(r.GetResume())
			jobAnalysis.Set(r.GetJob())
			rationales.Set(r.GetRationales())
			reloadVariants()
		}()
	})

	onSave := ui.UseEvent(func() {
		flash.Set("")
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			_, err = c.SaveSettings(ctx, &sitepb.Settings{OpenaiApiKey: apiKey.Get(), OpenaiModel: model.Get()})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("save failed")
				return
			}
			apiKey.Set("")
			flash.Set("settings saved")
			// Refresh key status + models with the new key.
			if s, err := c.GetSettings(ctx, &sitepb.Empty{}); err == nil {
				keySet.Set(s.GetKeySet())
			}
			if ml, err := c.ListModels(ctx, &sitepb.Empty{}); err == nil {
				models.Set(ml.GetModels())
			}
		}()
	})

	onReloadModels := ui.UseEvent(func() {
		flash.Set("loading models…")
		go func() {
			c, err := adminClient()
			if err != nil {
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			// If a key was just typed but not saved, store it first so the fetch uses it.
			if apiKey.Get() != "" {
				if _, err := c.SaveSettings(ctx, &sitepb.Settings{OpenaiApiKey: apiKey.Get()}); err != nil {
					if onAuthErr(err) {
						return
					}
					flash.Set("couldn't save the key")
					return
				}
				apiKey.Set("")
				keySet.Set(true)
			}
			ml, err := c.ListModels(ctx, &sitepb.Empty{})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("couldn't load models")
				return
			}
			models.Set(ml.GetModels())
			if len(ml.GetModels()) == 0 {
				flash.Set("no models returned — is the API key valid?")
			} else {
				flash.Set("models loaded — pick one and Save")
			}
		}()
	})

	onApply := ui.UseEvent(func() {
		t := tailored.Get()
		if t == nil {
			return
		}
		flash.Set("applying…")
		go func() {
			c, err := adminClient()
			if err != nil {
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if _, err := c.ApplyResume(ctx, t); err != nil {
				if onAuthErr(err) {
					return
				}
				flash.Set("apply failed")
				return
			}
			// The base résumé is permanent (diff baseline) — Apply only sets the active /resume.
			tailored.Set(nil)
			jobAnalysis.Set(nil)
			rationales.Set(nil)
			flash.Set("applied — this variant is now your active /resume (base résumé unchanged)")
		}()
	})
	onCancel := ui.UseEvent(func() {
		tailored.Set(nil)
		jobAnalysis.Set(nil)
		rationales.Set(nil)
		flash.Set("")
	})

	// selectVariant re-opens a saved variant for review / re-tweaking.
	selectVariant := func(meta *sitepb.TailoringMeta) {
		jobURL.Set(meta.GetJobUrl())
		flash.Set("loaded a saved variant — reanalyze to refresh it against the posting")
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			r, err := c.GetTailoring(ctx, &sitepb.TailoringId{Id: meta.GetId()})
			if onAuthErr(err) || err != nil || r.GetResume() == nil {
				return
			}
			tailored.Set(r.GetResume())
			jobAnalysis.Set(r.GetJob())
			rationales.Set(r.GetRationales())
		}()
	}
	// deleteVariant removes a saved variant and refreshes the list.
	deleteVariant := func(id int64) {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if _, err := c.DeleteTailoring(ctx, &sitepb.TailoringId{Id: id}); err != nil {
				if onAuthErr(err) {
					return
				}
				return
			}
			reloadVariants()
		}()
	}

	onSavePrompt := ui.UseEvent(func() {
		flash.Set("")
		go func() {
			c, err := adminClient()
			if err != nil {
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			ack, err := c.SavePrompt(ctx, &sitepb.PromptText{Text: promptText.Get()})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("save failed")
				return
			}
			flash.Set(ack.GetMessage())
		}()
	})
	onDryRun := ui.UseEvent(func() {
		flash.Set("")
		dryRun.Set(nil)
		dryRunning.Set(true)
		go func() {
			c, err := adminClient()
			if err != nil {
				dryRunning.Set(false)
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtxLong(token.Get())
			defer cancel()
			res, err := c.DryRunPrompt(ctx, &sitepb.PromptText{Text: promptText.Get()})
			dryRunning.Set(false)
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("dry run failed: " + err.Error())
				return
			}
			dryRun.Set(res)
		}()
	})
	onToggleSlack := ui.WrapHandler(func() { slackEnabled.Set(!slackEnabled.Get()) })
	onSaveSlack := ui.UseEvent(func() {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if _, err := c.SaveSlackConfig(ctx, &sitepb.SlackConfig{WebhookUrl: slackWebhook.Get(), Enabled: slackEnabled.Get(), PostHour: int32(slackHour.Get())}); err != nil {
				if onAuthErr(err) {
					return
				}
				flash.Set("save failed")
				return
			}
			slackWebhook.Set("")
			slackSet.Set(true)
			flash.Set("Slack config saved")
		}()
	})
	onPostNow := ui.UseEvent(func() {
		flash.Set("generating & posting…")
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtxLong(token.Get())
			defer cancel()
			ack, err := c.PostToSlackNow(ctx, &sitepb.Empty{})
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("post failed")
				return
			}
			flash.Set(ack.GetMessage())
		}()
	})
	// reloadPendingDevices re-fetches the pending-devices list — called after every approve/reject so
	// the resolved request drops off the list.
	reloadPendingDevices := func() {
		go func() {
			c, err := adminClient()
			if err != nil {
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			if list, err := c.ListCashFluxPendingDevices(ctx, &sitepb.Empty{}); err == nil {
				cashfluxPending.Set(list.GetItems())
			}
		}()
	}
	// setCashfluxBusy adds or removes deviceID from the in-flight set. It's a set (not a single
	// scalar) so approving/rejecting one device doesn't re-enable another device's Pair/Reject
	// buttons while that other request is still in flight — each row's busy state is independent.
	// Always builds a fresh map rather than mutating cashfluxBusy.Get() in place, matching this
	// file's convention of replacing state values wholesale (e.g. cashfluxPending.Set(...)).
	setCashfluxBusy := func(deviceID string, busy bool) {
		next := make(map[string]bool, len(cashfluxBusy.Get())+1)
		for id := range cashfluxBusy.Get() {
			next[id] = true
		}
		if busy {
			next[deviceID] = true
		} else {
			delete(next, deviceID)
		}
		cashfluxBusy.Set(next)
	}
	// onApprovePairing approves one pending device (row-owned handler — see the CashFlux hooks
	// gotcha: never call On* prop options inside a variable-length loop). On success it shows the
	// pairing code for the human cross-check; an already-resolved request (approved=false, no error)
	// just refreshes the list quietly.
	onApprovePairing := func(deviceID string, device *sitepb.CashFluxPendingDevice) {
		flash.Set("")
		setCashfluxBusy(deviceID, true)
		cashfluxCopied.Set(false)
		go func() {
			c, err := adminClient()
			if err != nil {
				setCashfluxBusy(deviceID, false)
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			resp, err := c.ApproveCashFluxPairing(ctx, &sitepb.CashFluxApprovePairingRequest{DeviceId: deviceID})
			setCashfluxBusy(deviceID, false)
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("approve failed: " + err.Error())
				return
			}
			if !resp.GetApproved() {
				flash.Set("that request was already resolved")
				reloadPendingDevices()
				return
			}
			cashfluxJustPaired.Set(device)
			cashfluxPairingCode.Set(resp.GetPairingCode())
			reloadPendingDevices()
		}()
	}
	// onRejectPairing declines one pending device.
	onRejectPairing := func(deviceID string) {
		flash.Set("")
		setCashfluxBusy(deviceID, true)
		go func() {
			c, err := adminClient()
			if err != nil {
				setCashfluxBusy(deviceID, false)
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			resp, err := c.RejectCashFluxPairing(ctx, &sitepb.CashFluxRejectPairingRequest{DeviceId: deviceID})
			setCashfluxBusy(deviceID, false)
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("reject failed: " + err.Error())
				return
			}
			if !resp.GetRejected() {
				flash.Set("that request was already resolved")
			}
			reloadPendingDevices()
		}()
	}
	onCopyPairingCode := ui.WrapHandler(func() {
		if code := cashfluxPairingCode.Get(); code != "" {
			copyToClipboard(code)
			cashfluxCopied.Set(true)
		}
	})
	// onMintActivationCode mints a fresh activation code for the owner account. Codes are single-use
	// and expire, so re-minting is always safe — the previous one simply goes unused.
	onMintActivationCode := ui.UseEvent(func() {
		if cashfluxActivationMinting.Get() {
			return
		}
		flash.Set("")
		cashfluxActivationMinting.Set(true)
		cashfluxActivationCopied.Set(false)
		go func() {
			c, err := adminClient()
			if err != nil {
				cashfluxActivationMinting.Set(false)
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			resp, err := c.MintCashFluxActivationCode(ctx, &sitepb.Empty{})
			cashfluxActivationMinting.Set(false)
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("couldn't generate a code: " + err.Error())
				return
			}
			cashfluxActivationCode.Set(resp.GetCode())
			cashfluxActivationExpires.Set(resp.GetExpiresAt())
		}()
	})
	onCopyActivationCode := ui.WrapHandler(func() {
		if code := cashfluxActivationCode.Get(); code != "" {
			copyToClipboard(code)
			cashfluxActivationCopied.Set(true)
		}
	})
	// onLoadMoreUsers fetches the next page of CashFlux users (offset = however many are already
	// shown) and appends it to the list — a single "load more" action rather than full pagination
	// controls, matching this deployment's expected scale (an admin-invited handful of accounts).
	onLoadMoreUsers := ui.UseEvent(func() {
		offset := len(cashfluxUsers.Get())
		cashfluxUsersLoading.Set(true)
		go func() {
			c, err := adminClient()
			if err != nil {
				cashfluxUsersLoading.Set(false)
				flash.Set("connection error")
				return
			}
			ctx, cancel := callCtx(token.Get())
			defer cancel()
			resp, err := c.ListCashFluxUsers(ctx, &sitepb.CashFluxListUsersRequest{Limit: cashfluxUsersPageSize, Offset: int32(offset)})
			cashfluxUsersLoading.Set(false)
			if onAuthErr(err) {
				return
			}
			if err != nil {
				flash.Set("couldn't load more users: " + err.Error())
				return
			}
			next := append(append([]*sitepb.CashFluxUser{}, cashfluxUsers.Get()...), resp.GetItems()...)
			cashfluxUsers.Set(next)
			cashfluxUsersMore.Set(len(resp.GetItems()) == cashfluxUsersPageSize)
		}()
	})

	var content ui.Node
	switch view.Get() {
	case "resume":
		content = resumeView(jobURL, onTailor, onApply, onCancel, canonical.Get(), tailored.Get(), jobAnalysis.Get(), rationales.Get(), variants.Get(), selectVariant, deleteVariant)
	case "settings":
		content = settingsView(keySet.Get(), models.Get(), model, apiKey, onSave, onReloadModels)
	case "rss":
		content = rssView(promptText, onSavePrompt, onDryRun, dryRunning.Get(), dryRun.Get(), slackWebhook, slackSet.Get(), slackEnabled.Get(), slackHour, onToggleSlack, onSaveSlack, onPostNow)
	case "cashflux":
		content = cashfluxView(cashfluxConfigured.Get(), activationCodeState{
			Code:      cashfluxActivationCode.Get(),
			ExpiresAt: cashfluxActivationExpires.Get(),
			Minting:   cashfluxActivationMinting.Get(),
			Copied:    cashfluxActivationCopied.Get(),
			OnMint:    onMintActivationCode,
			OnCopy:    onCopyActivationCode,
		}, cashfluxPending.Get(), cashfluxBusy.Get(), cashfluxJustPaired.Get(), cashfluxPairingCode.Get(), cashfluxCopied.Get(), onApprovePairing, onRejectPairing, onCopyPairingCode,
			cashfluxUsers.Get(), cashfluxUsersMore.Get(), cashfluxUsersLoading.Get(), onLoadMoreUsers, cashfluxStorage.Get())
	default:
		content = animeView(query, onSearch, onCheck, results.Get(), tracked.Get(), trackFn)
	}
	return consoleShell(view.Get(), navTo, onLogout, flash.Get(), content)
}
