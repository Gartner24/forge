package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables for a project",
}

var envSetCmd = &cobra.Command{
	Use:   "set <project> <KEY> <value>",
	Short: "Set an environment variable",
	Args:  cobra.ExactArgs(3),
	RunE:  runEnvSet,
}

var envGetCmd = &cobra.Command{
	Use:   "get <project> <KEY>",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvGet,
}

var envListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List environment variable keys",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvList,
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <project> <KEY>",
	Short: "Remove an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvUnset,
}

func init() {
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envUnsetCmd)
}

func envKey(project, key string) string {
	return "smeltforge." + project + "." + key
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	project, key, value := args[0], args[1], args[2]
	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Set(envKey(project, key), value); err != nil {
		return cmdErr(err)
	}
	printSuccess(fmt.Sprintf("set %s for project %s", key, project))
	return nil
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	project, key := args[0], args[1]
	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	val, err := store.Get(envKey(project, key))
	if err != nil {
		return cmdErr(err)
	}
	if isJSON() {
		printJSON(map[string]string{"key": key, "value": val})
	} else {
		fmt.Println(val)
	}
	return nil
}

func runEnvList(cmd *cobra.Command, args []string) error {
	project := args[0]
	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	allKeys, err := store.List()
	if err != nil {
		return cmdErr(err)
	}

	prefix := "smeltforge." + project + "."
	var keys []string
	for _, k := range allKeys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		shortKey := strings.TrimPrefix(k, prefix)
		if strings.HasPrefix(shortKey, "_") {
			continue
		}
		keys = append(keys, shortKey)
	}
	sort.Strings(keys)

	if isJSON() {
		if keys == nil {
			keys = []string{}
		}
		printJSON(map[string]any{"project": project, "keys": keys})
		return nil
	}

	if len(keys) == 0 {
		fmt.Printf("No env vars set for project %s.\n", project)
		return nil
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

func runEnvUnset(cmd *cobra.Command, args []string) error {
	project, key := args[0], args[1]
	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Delete(envKey(project, key)); err != nil {
		return cmdErr(err)
	}
	printSuccess(fmt.Sprintf("unset %s for project %s", key, project))
	return nil
}
