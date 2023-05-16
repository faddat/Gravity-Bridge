package cmd

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/cli"
)

const flagPassphrase = "passphrase"

// Commands registers a sub-tree of commands to interact with
// local private key storage.
func Commands(defaultNodeHome string) *cobra.Command {
	//nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "eth_keys",
		Short: "Manage your application's ethereum keys",
		Long: `Keyring management commands. Generated by the official Ethereum go library.

The keyring supports the following backends:
    test        Stores keys insecurely to disk. It does not prompt for a password to be unlocked
                and it should be use only for testing purposes.
`,
	}

	cmd.AddCommand(
		AddKeyCommand(),
	)

	cmd.PersistentFlags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.PersistentFlags().String(flags.FlagKeyringDir, "", "The client Keyring directory; if omitted, the default 'home' directory will be used")
	cmd.PersistentFlags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|test)")
	cmd.PersistentFlags().String(cli.OutputFlag, "text", "Output format (text|json)")

	return cmd
}

// AddKeyCommand defines a keys command to generate a key
func AddKeyCommand() *cobra.Command {
	//nolint: exhaustruct
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an encrypted private ethereum key",
		Long: `Derive a new private key and encrypt to disk.
`,
		RunE: runAddCmd,
	}

	cmd.Flags().String(flagPassphrase, "default", "Password used for ethereum key generation")
	cmd.Flags().Bool(flags.FlagDryRun, false, "Perform action, but don't add key to local keystore")

	cmd.SetOut(cmd.OutOrStdout())
	cmd.SetErr(cmd.ErrOrStderr())

	return cmd
}

type EthereumKeyOutput struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
}

func runAddCmd(cmd *cobra.Command, _ []string) error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting public key to ECDSA")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)

	keyOutput := EthereumKeyOutput{
		PrivateKey: hexutil.Encode(privateKeyBytes),
		PublicKey:  hexutil.Encode(publicKeyBytes),
		Address:    crypto.PubkeyToAddress(*publicKeyECDSA).Hex(),
	}

	if dryRun, errDryRun := cmd.Flags().GetBool(flags.FlagDryRun); !dryRun {
		if errDryRun != nil {
			fmt.Printf("ErrDryRun issue: %v", errDryRun)
		}
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		ks := keystore.NewKeyStore(clientCtx.KeyringDir, keystore.StandardScryptN, keystore.StandardScryptP)
		passphrase, err := cmd.Flags().GetString(flagPassphrase)
		if err != nil {
			return err
		}
		if _, err := ks.ImportECDSA(privateKey, passphrase); err != nil {
			return err
		}
	}

	return printCreate(cmd, keyOutput)
}

func printCreate(cmd *cobra.Command, keyOutput EthereumKeyOutput) error {
	output, errPrintCreate := cmd.Flags().GetString(cli.OutputFlag)
	if errPrintCreate != nil {
		panic(fmt.Sprintf("Out of bounds access for slice. Index passed: %v", errPrintCreate))
	}

	switch output {
	case keys.OutputFormatText:
		cmd.PrintErrln()
		cmd.Printf("private: %s \npublic: %s \naddress: %s\n", keyOutput.PrivateKey, keyOutput.PublicKey, keyOutput.Address)

	case keys.OutputFormatJSON:
		outputBytes, err := json.Marshal(keyOutput)
		if err != nil {
			return err
		}
		cmd.Println(string(outputBytes))

	default:
		return fmt.Errorf("invalid output format %s", output)
	}

	return nil
}
