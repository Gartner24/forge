package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	projects := reg.All()

	if isJSON() {
		if projects == nil {
			printJSON([]any{})
		} else {
			printJSON(projects)
		}
		return nil
	}

	if len(projects) == 0 {
		fmt.Println("No projects registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tSOURCE\tDOMAIN\tSTRATEGY")
	for _, p := range projects {
		src := p.Source.Type
		switch p.Source.Type {
		case "git":
			src = "git:" + p.Source.Repo
		case "registry":
			src = "registry:" + p.Source.Image
		case "local":
			src = "local:" + p.Source.Path
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, src, p.Domain, p.Strategy)
	}
	w.Flush()
	return nil
}
