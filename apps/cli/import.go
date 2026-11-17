package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/importer"
)

func importCmd() *cobra.Command {
	var output string
	var dryRun bool
	var format string

	cmd := &cobra.Command{
		Use:   "import <postman|insomnia> <file>",
		Short: "Convert a Postman collection or Insomnia export into request files",
		Long: `Read a Postman Collection (v2.1) or an Insomnia export (v4) and write one request file per
request, in the same layout the desktop app's import uses:

  <output>/<collection>/<folders>/<request>.yaml

Nothing existing is overwritten: a name that is already taken gets a -2, -3 ... suffix.
Use --dry-run to see the files that would be written.

Exit codes:
  0  the requests were converted (or listed, with --dry-run)
  1  some requests could not be written
  2  the file could not be read or is not a supported export`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, path := strings.ToLower(args[0]), args[1]
			if format != "text" && format != "json" {
				return fmt.Errorf("unknown --format %q (use text or json)", format)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}

			var name string
			var requests []importer.ImportedRequest
			switch kind {
			case "postman":
				name, requests, err = importer.ParsePostmanCollection(data)
			case "insomnia":
				name, requests, err = importer.ParseInsomniaExport(data)
			default:
				return fmt.Errorf("unknown source %q (use postman or insomnia)", args[0])
			}
			if err != nil {
				return err
			}
			if len(requests) == 0 {
				return fmt.Errorf("no requests found in %s", path)
			}

			res := importer.Save(output, name, requests, dryRun)

			if format == "json" {
				type jsonFile struct {
					Path   string `json:"path"`
					Title  string `json:"title"`
					Method string `json:"method"`
					URL    string `json:"url"`
				}
				doc := struct {
					Collection string     `json:"collection"`
					DryRun     bool       `json:"dryRun"`
					Files      []jsonFile `json:"files"`
					Skipped    int        `json:"skipped"`
				}{Collection: res.Collection, DryRun: dryRun, Files: []jsonFile{}, Skipped: res.Skipped}
				for _, s := range res.Saved {
					doc.Files = append(doc.Files, jsonFile{Path: s.Path, Title: s.Title, Method: s.Method, URL: s.URL})
				}
				b, err := json.MarshalIndent(doc, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
			} else {
				verb := "Wrote"
				if dryRun {
					verb = "Would write"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %d request file%s from %q:\n", verb, len(res.Saved), plural(len(res.Saved)), res.Collection)
				for _, s := range res.Saved {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-7s %s\n          %s\n", s.Method, s.URL, s.Path)
				}
				if res.Skipped > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "%d request%s could not be written\n", res.Skipped, plural(res.Skipped))
				}
			}

			if res.Skipped > 0 {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "requests", "Folder to write the request files into")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the files that would be written without writing them")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	return cmd
}
