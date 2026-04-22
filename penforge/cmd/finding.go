package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gartner24/forge/penforge/internal/engine"
	"github.com/gartner24/forge/penforge/internal/paths"
	"github.com/gartner24/forge/penforge/internal/scanner"
	"github.com/gartner24/forge/penforge/internal/store"
	"github.com/gartner24/forge/shared/registry"
	"github.com/spf13/cobra"
)

var findingCmd = &cobra.Command{
	Use:   "finding",
	Short: "Manage a finding's lifecycle",
}

var (
	findingAckReason    string
	findingAcceptReason string
)

var findingAckCmd = &cobra.Command{
	Use:   "acknowledge <id>",
	Short: "Acknowledge a finding (suppress re-alerts until severity increases)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFindingAcknowledge,
}

var findingAcceptCmd = &cobra.Command{
	Use:   "accept <id>",
	Short: "Permanently accept a finding risk (suppress alerts unless severity increases)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFindingAccept,
}

var findingVerifyCmd = &cobra.Command{
	Use:   "verify <id>",
	Short: "Mark a finding as fixed and trigger a re-scan",
	Args:  cobra.ExactArgs(1),
	RunE:  runFindingVerify,
}

func init() {
	findingAckCmd.Flags().StringVar(&findingAckReason, "reason", "", "Reason for acknowledging (required)")
	findingAckCmd.MarkFlagRequired("reason")

	findingAcceptCmd.Flags().StringVar(&findingAcceptReason, "reason", "", "Reason for accepting risk (required)")
	findingAcceptCmd.MarkFlagRequired("reason")

	findingCmd.AddCommand(findingAckCmd)
	findingCmd.AddCommand(findingAcceptCmd)
	findingCmd.AddCommand(findingVerifyCmd)
	rootCmd.AddCommand(findingCmd)
}

func runFindingAcknowledge(cmd *cobra.Command, args []string) error {
	return updateFindingState(args[0], store.StateAcknowledged, findingAckReason)
}

func runFindingAccept(cmd *cobra.Command, args []string) error {
	return updateFindingState(args[0], store.StateAccepted, findingAcceptReason)
}

func updateFindingState(id, state, reason string) error {
	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}
	auditFile, _ := paths.AuditFile()

	findingStore := store.NewFindingStoreWithAudit(findingsFile, auditFile)
	if err := findingStore.UpdateState(id, state, reason, time.Now().UTC()); err != nil {
		return cmdErr(err)
	}

	printSuccess(fmt.Sprintf("Finding %s marked as %s.", id, state))
	return nil
}

func runFindingVerify(cmd *cobra.Command, args []string) error {
	id := args[0]

	findingsFile, err := paths.FindingsFile()
	if err != nil {
		return cmdErr(err)
	}
	auditFile, _ := paths.AuditFile()
	findingStore := store.NewFindingStoreWithAudit(findingsFile, auditFile)

	f, err := findingStore.Get(id)
	if err != nil {
		return cmdErr(err)
	}

	if err := findingStore.UpdateState(id, store.StateFixed, "verification re-scan requested", time.Now().UTC()); err != nil {
		return cmdErr(err)
	}

	// Trigger a re-scan of the target.
	targetsFile, err := paths.TargetsFile()
	if err != nil {
		return cmdErr(err)
	}
	targets, err := registry.ReadScanTargets(targetsFile)
	if err != nil {
		return cmdErr(err)
	}

	var target *registry.ScanTarget
	for _, t := range targets {
		if t.ID == f.TargetID {
			cp := t
			target = &cp
			break
		}
	}
	if target == nil {
		printSuccess(fmt.Sprintf("Finding %s marked as fixed. Target not found for re-scan.", id))
		return nil
	}

	scansDir, err := paths.ScansDir()
	if err != nil {
		return cmdErr(err)
	}

	scanStore := store.NewScanStore(scansDir)
	engines := engine.ForTarget(*target)
	scanID := newScanID()

	if !isJSON() {
		fmt.Printf("Finding %s marked as fixed. Starting verification scan %s...\n", id, scanID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := scanner.RunScan(ctx, scanID, *target, engines, findingStore, scanStore, "", auditFile)
	if err != nil {
		scanner.MarkFailed(scanStore, scanID, err)
		return cmdErr(fmt.Errorf("verification scan failed: %w", err))
	}

	// If the finding is no longer present, mark it verified.
	updatedFinding, _ := findingStore.Get(id)
	if updatedFinding != nil && updatedFinding.State == store.StateFixed {
		_ = findingStore.UpdateState(id, store.StateVerified, "not seen in verification scan", time.Now().UTC())
		if isJSON() {
			printJSON(map[string]any{"finding_id": id, "state": "verified", "scan_id": scanID})
		} else {
			fmt.Printf("Verification scan complete. Finding %s is VERIFIED (not seen in re-scan).\n", id)
		}
		return nil
	}

	if isJSON() {
		printJSON(map[string]any{
			"finding_id":   id,
			"state":        "fixed",
			"scan_id":      scanID,
			"new_findings": result.NewFindings,
		})
	} else {
		fmt.Printf("Verification scan %s complete. Finding %s still in 'fixed' state -- may still be present.\n", scanID, id)
	}
	return nil
}
