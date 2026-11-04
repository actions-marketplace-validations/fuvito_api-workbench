package main

import "context"

type App struct {
	ctx              context.Context
	workspaceService *WorkspaceService
	requestService   *RequestService
	openapiService   *OpenAPIService
}

func NewApp() *App {
	ws := NewWorkspaceService()
	return &App{
		workspaceService: ws,
		requestService:   NewRequestService(ws),
		openapiService:   NewOpenAPIService(ws),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.workspaceService.setContext(ctx)
	a.requestService.setContext(ctx)
	a.openapiService.setContext(ctx)
}

func (a *App) shutdown(_ context.Context) {
	a.requestService.close()
}
