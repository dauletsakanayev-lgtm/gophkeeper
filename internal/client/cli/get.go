package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/crypto"
	"github.com/spf13/cobra"
)

var getOutput string

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Получить и расшифровать секрет",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		s, err := mustSession()
		if err != nil {
			return err
		}
		key, err := deriveMasterKey(s.Login)
		if err != nil {
			return err
		}
		secret, err := authClient(s).GetSecret(id)
		if err != nil {
			return fmt.Errorf("get secret: %w", err)
		}
		plaintext, err := crypto.Decrypt(key, secret.Data)
		if err != nil {
			return fmt.Errorf("decrypt (wrong master password?): %w", err)
		}

		fmt.Printf("id:       %s\n", secret.ID)
		fmt.Printf("type:     %s\n", secret.Type)
		fmt.Printf("meta:     %s\n", secret.Meta)
		fmt.Printf("revision: %d\n", secret.Revision)
		fmt.Println("---")

		switch secret.Type {
		case typeLogin:
			var p loginPayload
			if err := json.Unmarshal(plaintext, &p); err != nil {
				return fmt.Errorf("unmarshal login: %w", err)
			}
			fmt.Printf("login:    %s\npassword: %s\n", p.Login, p.Password)
		case typeText:
			var p textPayload
			if err := json.Unmarshal(plaintext, &p); err != nil {
				return fmt.Errorf("unmarshal text: %w", err)
			}
			fmt.Println(p.Content)
		case typeCard:
			var p cardPayload
			if err := json.Unmarshal(plaintext, &p); err != nil {
				return fmt.Errorf("unmarshal card: %w", err)
			}
			fmt.Printf("number:  %s\nexpires: %s\nholder:  %s\ncvv:     %s\n",
				p.Number, p.Expires, p.Holder, p.CVV)
		case typeBinary:
			if getOutput != "" {
				if err := os.WriteFile(getOutput, plaintext, 0o600); err != nil {
					return fmt.Errorf("write %s: %w", getOutput, err)
				}
				fmt.Printf("binary written to %s (%d bytes)\n", getOutput, len(plaintext))
			} else {
				// Пишем raw bytes в stdout (для pipe'ов).
				_, _ = os.Stdout.Write(plaintext)
			}
		default:
			fmt.Printf("(unknown type, raw plaintext, %d bytes)\n", len(plaintext))
		}
		return nil
	},
}

func init() {
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "",
		"для binary: путь к файлу вывода (по умолчанию stdout)")
}
