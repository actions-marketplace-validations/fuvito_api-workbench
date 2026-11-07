package main

import _ "embed"

//go:embed help/user-guide.md
var userGuideMarkdown string

type HelpService struct{}

func NewHelpService() *HelpService { return &HelpService{} }

func (h *HelpService) GetUserGuide() string { return userGuideMarkdown }
