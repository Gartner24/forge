package cmd

import (
	"fmt"

	"github.com/gartner24/forge/sparkforge/internal/model"
	"github.com/gartner24/forge/sparkforge/internal/router"
	"github.com/spf13/cobra"
)

var (
	sendTitle    string
	sendMessage  string
	sendPriority string
	sendSource   string
	sendChannel  string
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a notification to all matching channels",
	RunE:  runSend,
}

func init() {
	sendCmd.Flags().StringVar(&sendTitle, "title", "", "Notification title (required)")
	sendCmd.Flags().StringVar(&sendMessage, "message", "", "Notification body")
	sendCmd.Flags().StringVar(&sendPriority, "priority", "medium", "Priority: low, medium, high, critical")
	sendCmd.Flags().StringVar(&sendSource, "source", "sparkforge-cli", "Source identifier")
	sendCmd.Flags().StringVar(&sendChannel, "channel", "", "Send to a specific channel ID only")
	sendCmd.MarkFlagRequired("title")
}

func runSend(cmd *cobra.Command, args []string) error {
	priority, err := model.ParsePriority(sendPriority)
	if err != nil {
		return cmdErr(err)
	}

	msg := model.Message{
		Title:    sendTitle,
		Body:     sendMessage,
		Priority: priority,
		Source:   sendSource,
	}

	r, err := router.New()
	if err != nil {
		return cmdErr(fmt.Errorf("initialising router: %w", err))
	}

	var delivered []string

	if sendChannel != "" {
		delivered, err = r.SendToChannel(sendChannel, msg)
	} else {
		delivered, err = r.Send(msg)
	}
	if err != nil {
		return cmdErr(err)
	}

	if delivered == nil {
		delivered = []string{}
	}

	if isJSON() {
		printJSON(map[string]any{"ok": true, "channels": delivered})
		return nil
	}

	if len(delivered) == 0 {
		fmt.Println("Message not delivered to any channels (no channels matched priority threshold).")
	} else {
		fmt.Printf("Delivered to: %v\n", delivered)
	}
	return nil
}
