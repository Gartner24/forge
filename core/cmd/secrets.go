package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gartner24/forge/core/internal/paths"
	"github.com/gartner24/forge/shared/secrets"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage encrypted secrets",
}

var secretsSetSync bool
var secretsListPrefix string

var secretsSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Store a secret",
	Args:  cobra.ExactArgs(2),
	RunE:  runSecretsSet,
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Retrieve a secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsGet,
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secret keys",
	RunE:  runSecretsList,
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsDelete,
}

func init() {
	secretsSetCmd.Flags().BoolVar(&secretsSetSync, "sync", false, "Replicate to all FluxForge mesh nodes")
	secretsListCmd.Flags().StringVar(&secretsListPrefix, "prefix", "", "Filter keys by prefix")

	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
}

func openSecretsStore() (*secrets.Store, error) {
	if _, err := requireInit(); err != nil {
		return nil, err
	}
	p, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	return secrets.New(p)
}

func runSecretsSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	store, err := openSecretsStore()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Set(key, value); err != nil {
		return cmdErr(fmt.Errorf("setting secret: %w", err))
	}

	if secretsSetSync && verbose {
		fmt.Println("Note: --sync requires FluxForge. Secret stored locally only.")
	}

	printSuccess(fmt.Sprintf("secret %q stored", key))
	return nil
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	store, err := openSecretsStore()
	if err != nil {
		return cmdErr(err)
	}
	value, err := store.Get(key)
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		printJSON(map[string]string{"key": key, "value": value})
	} else {
		fmt.Println(value)
	}
	return nil
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	store, err := openSecretsStore()
	if err != nil {
		return cmdErr(err)
	}
	keys, err := store.List()
	if err != nil {
		return cmdErr(fmt.Errorf("listing secrets: %w", err))
	}

	// Filter by prefix.
	if secretsListPrefix != "" {
		filtered := keys[:0]
		for _, k := range keys {
			if strings.HasPrefix(k, secretsListPrefix) {
				filtered = append(filtered, k)
			}
		}
		keys = filtered
	}
	sort.Strings(keys)

	if isJSON() {
		if keys == nil {
			keys = []string{}
		}
		printJSON(map[string]any{"keys": keys})
		return nil
	}

	if len(keys) == 0 {
		fmt.Println("(no secrets)")
		return nil
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

func runSecretsDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	proceed, err := mustConfirm(fmt.Sprintf("Delete secret %q?", key))
	if err != nil {
		return cmdErr(err)
	}
	if !proceed {
		fmt.Println("Aborted.")
		return nil
	}

	store, err := openSecretsStore()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Delete(key); err != nil {
		return cmdErr(fmt.Errorf("deleting secret: %w", err))
	}

	printSuccess(fmt.Sprintf("secret %q deleted", key))
	return nil
}
