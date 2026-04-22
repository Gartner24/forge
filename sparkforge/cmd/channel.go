package cmd

import (
	"fmt"

	"github.com/gartner24/forge/shared/secrets"
	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/paths"
	"github.com/gartner24/forge/sparkforge/internal/registry"
	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage notification channels",
}

// channel add flags
var (
	chType        string
	chName        string
	chPriorityMin string
	chURL         string
	chToken       string
	chSMTPHost    string
	chSMTPPort    int
	chSMTPUser    string
	chSMTPPass    string
	chSMTPTo      string
)

var channelAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a notification channel",
	RunE:  runChannelAdd,
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all channels",
	RunE:  runChannelList,
}

var channelEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "Enable a channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelEnable,
}

var channelDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable a channel without deleting it",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelDisable,
}

var channelDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelDelete,
}

func init() {
	channelAddCmd.Flags().StringVar(&chType, "type", "", "Channel type: gotify, email, webhook (required)")
	channelAddCmd.Flags().StringVar(&chName, "name", "", "Channel name (required)")
	channelAddCmd.Flags().StringVar(&chPriorityMin, "priority-min", "low", "Minimum priority: low, medium, high, critical")
	channelAddCmd.Flags().StringVar(&chURL, "url", "", "URL (gotify server or webhook endpoint)")
	channelAddCmd.Flags().StringVar(&chToken, "token", "", "Gotify app token")
	channelAddCmd.Flags().StringVar(&chSMTPHost, "smtp-host", "", "SMTP host (email)")
	channelAddCmd.Flags().IntVar(&chSMTPPort, "smtp-port", 587, "SMTP port (email)")
	channelAddCmd.Flags().StringVar(&chSMTPUser, "smtp-user", "", "SMTP username (email)")
	channelAddCmd.Flags().StringVar(&chSMTPPass, "smtp-password", "", "SMTP password (email)")
	channelAddCmd.Flags().StringVar(&chSMTPTo, "to", "", "Recipient email address")
	channelAddCmd.MarkFlagRequired("type")
	channelAddCmd.MarkFlagRequired("name")

	channelCmd.AddCommand(channelAddCmd)
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelEnableCmd)
	channelCmd.AddCommand(channelDisableCmd)
	channelCmd.AddCommand(channelDeleteCmd)
}

func runChannelAdd(cmd *cobra.Command, args []string) error {
	ct := model.ChannelType(chType)
	if !ct.Valid() {
		return cmdErr(fmt.Errorf("invalid channel type %q: must be gotify, email, or webhook", chType))
	}

	pmin, err := model.ParsePriority(chPriorityMin)
	if err != nil {
		return cmdErr(err)
	}

	cfg := model.ChannelConfig{}
	switch ct {
	case model.ChannelTypeGotify:
		if chURL != "" {
			cfg.GotifyURL = chURL
		} else {
			cfg.GotifyURL = "http://localhost:7779"
		}
	case model.ChannelTypeEmail:
		if chSMTPHost == "" {
			return cmdErr(fmt.Errorf("--smtp-host is required for email channels"))
		}
		if chSMTPTo == "" {
			return cmdErr(fmt.Errorf("--to is required for email channels"))
		}
		cfg.SMTPHost = chSMTPHost
		cfg.SMTPPort = chSMTPPort
		cfg.SMTPUser = chSMTPUser
		cfg.SMTPTo = chSMTPTo
	case model.ChannelTypeWebhook:
		if chURL == "" {
			return cmdErr(fmt.Errorf("--url is required for webhook channels"))
		}
		cfg.WebhookURL = chURL
	}

	reg, err := registry.New()
	if err != nil {
		return cmdErr(err)
	}

	ch := model.Channel{
		Type:        ct,
		Name:        chName,
		Enabled:     true,
		PriorityMin: pmin,
		Config:      cfg,
	}

	added, err := reg.Add(ch)
	if err != nil {
		return cmdErr(err)
	}

	// Store sensitive credentials in secrets.
	if ct == model.ChannelTypeGotify && chToken != "" {
		if err := storeChannelSecret(added.ID, "gotify_token", chToken); err != nil {
			return cmdErr(fmt.Errorf("storing gotify token: %w", err))
		}
	}
	if ct == model.ChannelTypeEmail && chSMTPPass != "" {
		if err := storeChannelSecret(added.ID, "smtp_password", chSMTPPass); err != nil {
			return cmdErr(fmt.Errorf("storing smtp password: %w", err))
		}
	}

	if isJSON() {
		printJSON(added)
		return nil
	}
	printSuccess(fmt.Sprintf("channel %q added (id: %s)", added.Name, added.ID))
	return nil
}

func runChannelList(cmd *cobra.Command, args []string) error {
	reg, err := registry.New()
	if err != nil {
		return cmdErr(err)
	}
	channels, err := reg.List()
	if err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		if channels == nil {
			channels = []model.Channel{}
		}
		printJSON(channels)
		return nil
	}

	if len(channels) == 0 {
		fmt.Println("No channels configured. Run 'sparkforge channel add' to add one.")
		return nil
	}

	w := newTabWriter()
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tENABLED\tPRIORITY_MIN")
	for _, ch := range channels {
		enabled := "yes"
		if !ch.Enabled {
			enabled = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", ch.ID, ch.Name, ch.Type, enabled, ch.PriorityMin)
	}
	w.Flush()
	return nil
}

func runChannelEnable(cmd *cobra.Command, args []string) error {
	return setChannelEnabled(args[0], true)
}

func runChannelDisable(cmd *cobra.Command, args []string) error {
	return setChannelEnabled(args[0], false)
}

func runChannelDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	reg, err := registry.New()
	if err != nil {
		return cmdErr(err)
	}

	ch, err := reg.Get(id)
	if err != nil {
		return cmdErr(err)
	}

	ok, err := mustConfirm(fmt.Sprintf("Delete channel %q (%s)?", ch.Name, id))
	if err != nil {
		return cmdErr(err)
	}
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := reg.Delete(id); err != nil {
		return cmdErr(err)
	}

	// Clean up any stored secrets for this channel.
	secretsPath, _ := paths.SecretsFile()
	if sec, err := secrets.New(secretsPath); err == nil {
		_ = sec.Delete(fmt.Sprintf("sparkforge.channels.%s.gotify_token", id))
		_ = sec.Delete(fmt.Sprintf("sparkforge.channels.%s.smtp_password", id))
	}

	printSuccess(fmt.Sprintf("channel %q deleted", ch.Name))
	return nil
}

func setChannelEnabled(id string, enabled bool) error {
	reg, err := registry.New()
	if err != nil {
		return cmdErr(err)
	}
	if err := reg.SetEnabled(id, enabled); err != nil {
		return cmdErr(err)
	}
	state := "enabled"
	if !enabled {
		state = "disabled"
	}
	printSuccess(fmt.Sprintf("channel %s %s", id, state))
	return nil
}

func storeChannelSecret(channelID, field, value string) error {
	secretsPath, err := paths.SecretsFile()
	if err != nil {
		return err
	}
	sec, err := secrets.New(secretsPath)
	if err != nil {
		return err
	}
	return sec.Set(fmt.Sprintf("sparkforge.channels.%s.%s", channelID, field), value)
}
