package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// configCommands are the commands that read the project file, in the order config show lists them.
var configCommands = []string{"conform", "run", "validate"}

func newConfigTarget(name string) *cobra.Command {
	switch name {
	case "conform":
		return conformCmd()
	case "run":
		return runCmd()
	default:
		return validateCmd()
	}
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Look at the project file (clrsig.yaml)",
	}
	cmd.AddCommand(configShowCmd())
	return cmd
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [command]",
		Short: "Show what conform, run and validate take from the project file",
		Long: `Show, for conform, run and validate (or one of them), each setting the project file can supply, its
value, and where the value comes from: the file, or the built-in default.

This is what the command uses when you give no flag and no argument. A flag or argument on the
command line always replaces the value shown. --config and --no-config work as they do for the
commands themselves.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names := configCommands
			if len(args) == 1 {
				ok := false
				for _, n := range configCommands {
					ok = ok || n == args[0]
				}
				if !ok {
					return fmt.Errorf("config show supports: %s", strings.Join(configCommands, ", "))
				}
				names = args[:1]
			}
			cfg, ignored, cwd, err := findProjectConfig(cmd)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			switch {
			case ignored:
				fmt.Fprintln(w, "The project file is ignored (--no-config): every command uses its built-in defaults.")
				return nil
			case cfg == nil:
				fmt.Fprintf(w, "No project file found from %s upward. clrsig init creates one.\n", cwd)
				return nil
			}
			fmt.Fprintf(w, "Project file: %s\n", displayPath(cfg.Path(), cwd))
			for _, name := range names {
				if err := showCommandConfig(cmd, name); err != nil {
					return err
				}
			}
			fmt.Fprintln(w, "\nA flag or argument on the command line replaces any value above.")
			return nil
		},
	}
}

// showCommandConfig applies the project file to a fresh copy of the command, exactly as the
// command does when it runs, and prints what ended up in each flag and argument.
func showCommandConfig(from *cobra.Command, name string) error {
	root := &cobra.Command{Use: "clrsig"}
	addConfigFlags(root)
	target := newConfigTarget(name)
	root.AddCommand(target)
	for _, flag := range []string{"config", "no-config"} {
		if err := root.PersistentFlags().Set(flag, from.Flag(flag).Value.String()); err != nil {
			return err
		}
	}
	args, lc, err := loadAndApply(target, nil)
	if err != nil {
		return err
	}
	used := map[string]bool{}
	if lc != nil {
		for _, k := range lc.used {
			used[k] = true
		}
	}

	type row struct{ key, value, source string }
	var rows []row
	for _, s := range configSettings {
		if !s.appliesTo(name) {
			continue
		}
		r := row{key: s.key, value: "(not set)", source: "default"}
		switch {
		case used[s.key] && s.flag == "":
			r.value, r.source = strings.Join(args, ", "), "file"
		case used[s.key]:
			r.value, r.source = flagValue(target, s.flag), "file"
		case s.flag != "":
			if d := flagDefault(target, s.flag); d != "" {
				r.value = d
			}
		}
		rows = append(rows, r)
	}
	w := from.OutOrStdout()
	fmt.Fprintf(w, "\nclrsig %s (with no flags or arguments):\n", name)
	width := 0
	for _, r := range rows {
		width = max(width, len(r.key))
	}
	valueWidth := 0
	for _, r := range rows {
		valueWidth = max(valueWidth, len(r.value))
	}
	for _, r := range rows {
		fmt.Fprintf(w, "  %-*s  %-*s  %s\n", width, r.key, valueWidth, r.value, r.source)
	}
	return nil
}

func flagValue(cmd *cobra.Command, name string) string {
	f := cmd.Flags().Lookup(name)
	if f == nil {
		return ""
	}
	return cleanList(f.Value.String())
}

func flagDefault(cmd *cobra.Command, name string) string {
	f := cmd.Flags().Lookup(name)
	if f == nil {
		return ""
	}
	return cleanList(f.DefValue)
}

// cleanList turns the "[a,b]" a list flag prints into a, b, and an empty list into nothing.
func cleanList(v string) string {
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
		v = strings.Join(strings.Split(strings.Trim(v, "[]"), ","), ", ")
		v = strings.TrimSpace(v)
	}
	return v
}
