package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx               context.Context
	workspaceService  *WorkspaceService
	requestService    *RequestService
	openapiService    *OpenAPIService
	importService     *ImportService
	bundleService     *BundleService
	settingsService   *SettingsService
	demoService       *DemoService
	conformanceServer *ConformanceServerService
	scenarioService   *ScenarioService
	helpService       *HelpService
	cliBuilder        *CliBuilderService
	conformance       *ConformanceService
	hasDirtyTabs      bool
}

func NewApp() *App {
	ws := NewWorkspaceService()
	ss := NewSettingsService()
	rs := NewRequestService(ws, ss)
	ss.onSave = rs.RebuildExecutor // rebuild HTTP client whenever settings are saved
	return &App{
		workspaceService:  ws,
		requestService:    rs,
		openapiService:    NewOpenAPIService(ws),
		importService:     NewImportService(ws),
		bundleService:     NewBundleService(ws),
		settingsService:   ss,
		demoService:       NewDemoService(ws),
		conformanceServer: NewConformanceServerService(),
		scenarioService:   NewScenarioService(ws, ss),
		helpService:       NewHelpService(),
		cliBuilder:        NewCliBuilderService(ws),
		conformance:       NewConformanceService(ws, ss),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.workspaceService.setContext(ctx)
	a.requestService.setContext(ctx)
	a.openapiService.setContext(ctx)
	a.importService.setContext(ctx)
	a.bundleService.setContext(ctx)
	a.settingsService.setContext(ctx)
	a.demoService.setContext(ctx)
	a.scenarioService.setContext(ctx)
	a.cliBuilder.setContext(ctx)
	a.conformance.setContext(ctx)
	// Set title now that Wails context is live (workspace may already be set from config).
	a.workspaceService.syncTitle()
}

func (a *App) shutdown(_ context.Context) {
	a.requestService.close()
	a.demoService.StopDemo()
	a.conformanceServer.Stop()
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
	// Window is still alive here — safe to read size.
	w, h := runtime.WindowGetSize(ctx)
	if w >= 900 && h >= 600 {
		cfg := loadConfig()
		cfg.WindowWidth = w
		cfg.WindowHeight = h
		saveConfig(cfg)
	}

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
