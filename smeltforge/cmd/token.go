package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gartner24/forge/smeltforge/internal/registry"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage CI deploy tokens",
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a CI token for a project",
	RunE:  runTokenCreate,
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List CI tokens for a project",
	RunE:  runTokenList,
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke <token-id>",
	Short: "Revoke a CI token",
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenRevoke,
}

var tokenLabel string

func init() {
	tokenCreateCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	tokenCreateCmd.Flags().StringVar(&tokenLabel, "label", "", "Human-readable label")
	tokenCreateCmd.MarkFlagRequired("project")

	tokenListCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	tokenListCmd.MarkFlagRequired("project")

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)
}

func ciTokenKey(project, tokenID string) string {
	return "smeltforge." + project + "._citoken_" + tokenID
}

func runTokenCreate(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}

	tokenID, err := generateSecret()
	if err != nil {
		return cmdErr(err)
	}
	tokenValue, err := generateSecret()
	if err != nil {
		return cmdErr(err)
	}

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Set(ciTokenKey(projectID, tokenID), tokenValue); err != nil {
		return cmdErr(err)
	}

	p.CITokens = append(p.CITokens, registry.CIToken{ID: tokenID, Label: tokenLabel})
	if err := reg.Update(p); err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		printJSON(map[string]string{"id": tokenID, "token": tokenValue})
	} else {
		fmt.Printf("token ID: %s\ntoken:    %s\n\nStore the token value now -- it cannot be retrieved later.\n", tokenID, tokenValue)
	}
	return nil
}

func runTokenList(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	p, err := reg.Get(projectID)
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		tokens := p.CITokens
		if tokens == nil {
			tokens = []registry.CIToken{}
		}
		printJSON(tokens)
		return nil
	}

	if len(p.CITokens) == 0 {
		fmt.Printf("No tokens for project %s.\n", projectID)
		return nil
	}
	for _, t := range p.CITokens {
		fmt.Printf("%s  %s\n", t.ID[:8]+"...", t.Label)
	}
	return nil
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	tokenID := args[0]

	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}

	// Find which project owns this token.
	var ownerProject string
	for _, p := range reg.All() {
		for _, t := range p.CITokens {
			if t.ID == tokenID {
				ownerProject = p.ID
			}
		}
	}
	if ownerProject == "" {
		return cmdErr(fmt.Errorf("token %q not found", tokenID))
	}

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	_ = store.Delete(ciTokenKey(ownerProject, tokenID))

	p, _ := reg.Get(ownerProject)
	tokens := p.CITokens[:0]
	for _, t := range p.CITokens {
		if t.ID != tokenID {
			tokens = append(tokens, t)
		}
	}
	p.CITokens = tokens
	if err := reg.Update(p); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("token %s revoked", tokenID[:8]+"..."))
	return nil
}
