package main

import "context"

type App struct {
	ctx              context.Context
	workspaceService *WorkspaceService
	requestService   *RequestService
	openapiService   *OpenAPIService
	settingsService  *SettingsService
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
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.workspaceService.setContext(ctx)
	a.requestService.setContext(ctx)
	a.openapiService.setContext(ctx)
	a.settingsService.setContext(ctx)
}

func (a *App) shutdown(_ context.Context) {
	a.requestService.close()
}
