package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhook secrets",
}

var webhookShowCmd = &cobra.Command{
	Use:   "show <project>",
	Short: "Print the current webhook URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookShow,
}

var webhookRegenerateCmd = &cobra.Command{
	Use:   "regenerate <project>",
	Short: "Rotate the webhook secret",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookRegenerate,
}

func init() {
	webhookCmd.AddCommand(webhookShowCmd)
	webhookCmd.AddCommand(webhookRegenerateCmd)
}

func webhookSecretKey(project string) string {
	return "smeltforge." + project + "._webhook"
}

func runWebhookShow(cmd *cobra.Command, args []string) error {
	project := args[0]

	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	p, err := reg.Get(project)
	if err != nil {
		return cmdErr(err)
	}

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}

	secret, err := store.Get(webhookSecretKey(project))
	if err != nil {
		return cmdErr(fmt.Errorf("no webhook secret for %s -- run: smeltforge webhook regenerate %s", project, project))
	}

	url := fmt.Sprintf("https://%s/_smeltforge/webhook/%s/%s", p.Domain, project, secret)

	if isJSON() {
		printJSON(map[string]string{"project": project, "url": url})
	} else {
		fmt.Println(url)
	}
	return nil
}

func runWebhookRegenerate(cmd *cobra.Command, args []string) error {
	project := args[0]

	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	if _, err := reg.Get(project); err != nil {
		return cmdErr(err)
	}

	secret, err := generateSecret()
	if err != nil {
		return cmdErr(fmt.Errorf("generating secret: %w", err))
	}

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Set(webhookSecretKey(project), secret); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("webhook secret regenerated for project %s", project))
	return nil
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
