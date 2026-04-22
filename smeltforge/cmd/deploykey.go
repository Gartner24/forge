package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var deployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "Manage SSH deploy keys for git source projects",
}

var deployKeyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new SSH deploy key pair",
	RunE:  runDeployKeyGenerate,
}

var deployKeyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the public deploy key",
	RunE:  runDeployKeyShow,
}

var deployKeyRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Generate a new deploy key pair (replacing the existing one)",
	RunE:  runDeployKeyGenerate,
}

func init() {
	deployKeyGenerateCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	deployKeyGenerateCmd.MarkFlagRequired("project")

	deployKeyShowCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	deployKeyShowCmd.MarkFlagRequired("project")

	deployKeyRotateCmd.Flags().StringVar(&projectID, "project", "", "Project ID (required)")
	deployKeyRotateCmd.MarkFlagRequired("project")

	deployKeyCmd.AddCommand(deployKeyGenerateCmd)
	deployKeyCmd.AddCommand(deployKeyShowCmd)
	deployKeyCmd.AddCommand(deployKeyRotateCmd)
}

func deployKeySecretKey(project string) string {
	return "smeltforge." + project + "._deploykey"
}

func runDeployKeyGenerate(cmd *cobra.Command, args []string) error {
	reg, err := loadRegistry()
	if err != nil {
		return cmdErr(err)
	}
	if _, err := reg.Get(projectID); err != nil {
		return cmdErr(err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return cmdErr(fmt.Errorf("generating key: %w", err))
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return cmdErr(fmt.Errorf("creating public key: %w", err))
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(pubKey)

	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}
	if err := store.Set(deployKeySecretKey(projectID), string(privPEM)); err != nil {
		return cmdErr(err)
	}

	if isJSON() {
		printJSON(map[string]string{"public_key": string(pubKeyBytes)})
	} else {
		fmt.Printf("Public key (add to GitHub -> Settings -> Deploy keys):\n\n%s\n", string(pubKeyBytes))
		fmt.Printf("Private key stored in forge secrets as %s\n", deployKeySecretKey(projectID))
	}
	return nil
}

func runDeployKeyShow(cmd *cobra.Command, args []string) error {
	store, err := loadSecrets()
	if err != nil {
		return cmdErr(err)
	}

	privPEM, err := store.Get(deployKeySecretKey(projectID))
	if err != nil {
		return cmdErr(fmt.Errorf("no deploy key for %s -- run: smeltforge deploy-key generate --project %s", projectID, projectID))
	}

	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return cmdErr(fmt.Errorf("invalid stored key"))
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return cmdErr(fmt.Errorf("parsing key: %w", err))
	}

	pubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return cmdErr(err)
	}

	pubBytes := ssh.MarshalAuthorizedKey(pubKey)
	if isJSON() {
		printJSON(map[string]string{"public_key": string(pubBytes)})
	} else {
		fmt.Print(string(pubBytes))
	}
	return nil
}
