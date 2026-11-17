package clibuild

import (
	"regexp"
	"strings"
)

// Shell is a target command-line shell.
type Shell string

const (
	Bash       Shell = "bash"
	PowerShell Shell = "powershell"
	Cmd        Shell = "cmd"
)

// Shells lists the supported shells in display order.
var Shells = []Shell{Bash, PowerShell, Cmd}

var (
	safeBash = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)
	// In PowerShell a comma builds an array, @ starts a splat and % is an alias, so none are safe bare.
	safePS  = regexp.MustCompile(`^[A-Za-z0-9_+=:./\\-]+$`)
	safeCmd = regexp.MustCompile(`^[A-Za-z0-9_@+=:./\\-]+$`)
)

// quote returns s quoted for shell, and a warning when the shell may still alter it.
func quote(shell Shell, s string) (string, string) {
	switch shell {
	case PowerShell:
		if safePS.MatchString(s) {
			return s, ""
		}
		return "'" + strings.ReplaceAll(s, "'", "''") + "'", ""
	case Cmd:
		if safeCmd.MatchString(s) {
			return s, ""
		}
		warn := ""
		if strings.ContainsAny(s, "%") {
			warn = "contains %, which cmd.exe may expand as a variable; use PowerShell or double the % in a .bat file"
		}
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`, warn
	default:
		if safeBash.MatchString(s) {
			return s, ""
		}
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'", ""
	}
}

func continuation(shell Shell) string {
	switch shell {
	case PowerShell:
		return " `"
	case Cmd:
		return " ^"
	}
	return " \\"
}
