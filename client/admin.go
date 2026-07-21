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

// AdminApp is the root admin component: a login gate, then the console (anime / résumé / settings).
func AdminApp() ui.Node {
	token := ui.UseState(loadToken())
	view := ui.UseState("anime")
	flash := ui.UseState("")

	username := ui.UseState("")
	password := ui.UseState("")

	query := ui.UseState("")
	results := ui.UseState[[]*sitepb.Anime](nil)
	tracked := ui.UseState[[]*sitepb.Anime](nil)

	jobURL := ui.UseState("")
	tailored := ui.UseState[*sitepb.Resume](nil)

	keySet := ui.UseState(false)
	model := ui.UseState("")
	models := ui.UseState[[]string](nil)
	apiKey := ui.UseState("")

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
		}
		return nil
	}, authed, view.Get())

	if !authed {
		onLogin := ui.UseEvent(func() {
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
		return loginView(username, password, onLogin, flash.Get())
	}

	// Console handlers.
	onLogout := ui.UseEvent(func() { clearToken(); token.Set(""); results.Set(nil); tracked.Set(nil) })
	navTo := func(v string) ui.Handler { return ui.WrapHandler(func() { flash.Set(""); view.Set(v) }) }

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
			flash.Set("")
			tailored.Set(r)
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

	var content ui.Node
	switch view.Get() {
	case "resume":
		content = resumeView(jobURL, onTailor, tailored.Get())
	case "settings":
		content = settingsView(keySet.Get(), models.Get(), model, apiKey, onSave, onReloadModels)
	default:
		content = animeView(query, onSearch, onCheck, results.Get(), tracked.Get(), trackFn)
	}
	return consoleShell(view.Get(), navTo, onLogout, flash.Get(), content)
}
