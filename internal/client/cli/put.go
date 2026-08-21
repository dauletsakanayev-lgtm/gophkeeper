package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/crypto"
	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Создать новый секрет (login/text/binary/card)",
}

var (
	putMeta        string
	putLoginLogin  string
	putLoginPass   string
	putTextContent string
	putBinaryFile  string
	putCardNumber  string
	putCardExpires string
	putCardHolder  string
	putCardCVV     string
)

var putLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Сохранить пару логин/пароль",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if putLoginLogin == "" || putLoginPass == "" {
			return fmt.Errorf("--login and --password are required")
		}
		payload, err := json.Marshal(loginPayload{Login: putLoginLogin, Password: putLoginPass})
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		return putEncrypted(typeLogin, payload, putMeta)
	},
}

var putTextCmd = &cobra.Command{
	Use:   "text",
	Short: "Сохранить произвольный текст",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if putTextContent == "" {
			return fmt.Errorf("--content is required")
		}
		payload, err := json.Marshal(textPayload{Content: putTextContent})
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		return putEncrypted(typeText, payload, putMeta)
	},
}

var putBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "Сохранить бинарные данные из файла",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if putBinaryFile == "" {
			return fmt.Errorf("--file is required")
		}
		content, err := os.ReadFile(putBinaryFile)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		return putEncrypted(typeBinary, content, putMeta)
	},
}

var putCardCmd = &cobra.Command{
	Use:   "card",
	Short: "Сохранить данные банковской карты",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if putCardNumber == "" || putCardExpires == "" {
			return fmt.Errorf("--number and --expires are required")
		}
		payload, err := json.Marshal(cardPayload{
			Number:  putCardNumber,
			Expires: putCardExpires,
			Holder:  putCardHolder,
			CVV:     putCardCVV,
		})
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		return putEncrypted(typeCard, payload, putMeta)
	},
}

// putEncrypted — общий флоу put'ов: получить key → зашифровать payload → API create.
func putEncrypted(secretType string, plaintext []byte, meta string) error {
	s, err := mustSession()
	if err != nil {
		return err
	}
	key, err := deriveMasterKey(s.Login)
	if err != nil {
		return err
	}
	sealed, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	secret, err := authClient(s).CreateSecret(secretType, sealed, meta)
	if err != nil {
		return fmt.Errorf("create secret: %w", err)
	}
	fmt.Printf("created secret %s (type=%s revision=%d)\n", secret.ID, secret.Type, secret.Revision)
	return nil
}

func init() {
	putCmd.PersistentFlags().StringVarP(&putMeta, "meta", "m", "", "произвольная метаинформация")

	putLoginCmd.Flags().StringVar(&putLoginLogin, "login", "", "логин")
	putLoginCmd.Flags().StringVar(&putLoginPass, "password", "", "пароль")

	putTextCmd.Flags().StringVar(&putTextContent, "content", "", "текстовое содержимое")

	putBinaryCmd.Flags().StringVar(&putBinaryFile, "file", "", "путь к файлу")

	putCardCmd.Flags().StringVar(&putCardNumber, "number", "", "номер карты")
	putCardCmd.Flags().StringVar(&putCardExpires, "expires", "", "срок MM/YY")
	putCardCmd.Flags().StringVar(&putCardHolder, "holder", "", "имя держателя")
	putCardCmd.Flags().StringVar(&putCardCVV, "cvv", "", "CVV")

	putCmd.AddCommand(putLoginCmd, putTextCmd, putBinaryCmd, putCardCmd)
}
