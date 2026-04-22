package cmd

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens",
}

var tokenName string

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API token",
	RunE:  runTokenCreate,
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke <name>",
	Short: "Revoke an API token by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenRevoke,
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API token names",
	RunE:  runTokenList,
}

func init() {
	tokenCreateCmd.Flags().StringVar(&tokenName, "name", "", "Token name (required)")
	tokenCreateCmd.MarkFlagRequired("name")

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)
	tokenCmd.AddCommand(tokenListCmd)
}

func runTokenCreate(cmd *cobra.Command, args []string) error {
	sec, err := openSecrets()
	if err != nil {
		return cmdErr(err)
	}

	key := "sparkforge.api_tokens." + tokenName
	existing, _ := sec.Get(key)
	if existing != "" {
		return cmdErr(fmt.Errorf("token %q already exists -- revoke it first", tokenName))
	}

	token := generateToken()
	if err := sec.Set(key, token); err != nil {
		return cmdErr(fmt.Errorf("storing token: %w", err))
	}

	if isJSON() {
		printJSON(map[string]string{"name": tokenName, "token": token})
		return nil
	}
	fmt.Printf("Token created. Store it now -- it will not be shown again.\n\nName:  %s\nToken: %s\n", tokenName, token)
	return nil
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	name := args[0]
	sec, err := openSecrets()
	if err != nil {
		return cmdErr(err)
	}

	key := "sparkforge.api_tokens." + name
	if _, err := sec.Get(key); err != nil {
		return cmdErr(fmt.Errorf("token %q not found", name))
	}

	if err := sec.Delete(key); err != nil {
		return cmdErr(fmt.Errorf("revoking token: %w", err))
	}

	printSuccess(fmt.Sprintf("token %q revoked", name))
	return nil
}

func runTokenList(cmd *cobra.Command, args []string) error {
	sec, err := openSecrets()
	if err != nil {
		return cmdErr(err)
	}

	keys, err := sec.List()
	if err != nil {
		return cmdErr(err)
	}

	const prefix = "sparkforge.api_tokens."
	var names []string
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			names = append(names, strings.TrimPrefix(k, prefix))
		}
	}

	if isJSON() {
		if names == nil {
			names = []string{}
		}
		printJSON(names)
		return nil
	}

	if len(names) == 0 {
		fmt.Println("No API tokens. Run 'sparkforge token create --name <n>' to create one.")
		return nil
	}
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

func openSecrets() (*secrets.Store, error) {
	p, err := paths.SecretsFile()
	if err != nil {
		return nil, err
	}
	return secrets.New(p)
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("%x", b)
}
