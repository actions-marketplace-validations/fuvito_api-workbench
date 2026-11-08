package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx              context.Context
	workspaceService *WorkspaceService
	requestService   *RequestService
	openapiService   *OpenAPIService
	settingsService  *SettingsService
	demoService      *DemoService
	scenarioService  *ScenarioService
	helpService      *HelpService
	hasDirtyTabs     bool
}

func NewApp() *App {
	ws := NewWorkspaceService()
	ss := NewSettingsService()
	rs := NewRequestService(ws, ss)
	ss.onSave = rs.RebuildExecutor // rebuild HTTP client whenever settings are saved
	return &App{
		workspaceService: ws,
		requestService:   rs,
		openapiService:   NewOpenAPIService(ws),
		settingsService:  ss,
		demoService:      NewDemoService(ws),
		scenarioService:  NewScenarioService(ws, ss),
		helpService:      NewHelpService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.workspaceService.setContext(ctx)
	a.requestService.setContext(ctx)
	a.openapiService.setContext(ctx)
	a.settingsService.setContext(ctx)
	a.demoService.setContext(ctx)
	a.scenarioService.setContext(ctx)
}

func (a *App) shutdown(_ context.Context) {
	a.requestService.close()
	a.demoService.StopDemo()
}

// SetHasDirtyTabs is called by the frontend whenever the overall dirty state changes.
func (a *App) SetHasDirtyTabs(dirty bool) {
	a.hasDirtyTabs = dirty
}

// Exit is called by the File > Exit menu item. OnBeforeClose handles the dirty-tab check.
func (a *App) Exit() {
	runtime.Quit(a.ctx)
}

// beforeClose is registered as OnBeforeClose; returns true to cancel the close.
func (a *App) beforeClose(ctx context.Context) bool {
	if !a.hasDirtyTabs {
		return false
	}
	result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Unsaved changes",
		Message:       "You have unsaved changes that will be lost. Exit anyway?",
		Buttons:       []string{"Exit", "Cancel"},
		DefaultButton: "Cancel",
		CancelButton:  "Cancel",
	})
	if err != nil {
		return false // allow close if dialog fails
	}
	// On Windows the native QuestionDialog returns "Yes"/"No" regardless of custom
	// button labels; on macOS it returns the label text. Cancel the close only when
	// the user explicitly declined — anything else (Yes, Exit, unknown) allows exit.
	return result == "Cancel" || result == "No"
}
