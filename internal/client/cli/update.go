package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/crypto"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Обновить существующий секрет по id",
}

// Флаги — те же что у put, будем читать их из общих переменных putLogin/putText/etc.

var updateLoginCmd = &cobra.Command{
	Use:   "login <id>",
	Short: "Обновить секрет типа login",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if putLoginLogin == "" || putLoginPass == "" {
			return fmt.Errorf("--login and --password are required")
		}
		payload, _ := json.Marshal(loginPayload{Login: putLoginLogin, Password: putLoginPass})
		return updateEncrypted(args[0], typeLogin, payload, putMeta)
	},
}

var updateTextCmd = &cobra.Command{
	Use:   "text <id>",
	Short: "Обновить секрет типа text",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if putTextContent == "" {
			return fmt.Errorf("--content is required")
		}
		payload, _ := json.Marshal(textPayload{Content: putTextContent})
		return updateEncrypted(args[0], typeText, payload, putMeta)
	},
}

var updateBinaryCmd = &cobra.Command{
	Use:   "binary <id>",
	Short: "Обновить секрет типа binary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if putBinaryFile == "" {
			return fmt.Errorf("--file is required")
		}
		content, err := os.ReadFile(putBinaryFile)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		return updateEncrypted(args[0], typeBinary, content, putMeta)
	},
}

var updateCardCmd = &cobra.Command{
	Use:   "card <id>",
	Short: "Обновить секрет типа card",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if putCardNumber == "" || putCardExpires == "" {
			return fmt.Errorf("--number and --expires are required")
		}
		payload, _ := json.Marshal(cardPayload{
			Number: putCardNumber, Expires: putCardExpires,
			Holder: putCardHolder, CVV: putCardCVV,
		})
		return updateEncrypted(args[0], typeCard, payload, putMeta)
	},
}

// updateEncrypted — общий флоу update: получить key → узнать текущий revision →
// зашифровать payload → PUT. При 409 (revision mismatch) — интерактивный resolver.
func updateEncrypted(id, secretType string, plaintext []byte, meta string) error {
	s, err := mustSession()
	if err != nil {
		return err
	}
	key, err := deriveMasterKey(s.Login)
	if err != nil {
		return err
	}
	client := authClient(s)

	// 1. Получаем актуальный секрет — нужна текущая revision.
	current, err := client.GetSecret(id)
	if err != nil {
		return fmt.Errorf("get current secret: %w", err)
	}

	// 2. Шифруем новый payload.
	sealed, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	// 3. Пробуем обновить.
	updated, err := client.UpdateSecret(id, sealed, meta, current.Revision)
	switch {
	case err == nil:
		fmt.Printf("updated secret %s (revision %d -> %d)\n", updated.ID, current.Revision, updated.Revision)
		return nil
	case errors.Is(err, api.ErrConflict):
		// Другой клиент опередил — updated содержит АКТУАЛЬНЫЙ секрет с сервера.
		return resolveConflict(client, key, id, current.Type, current.Meta, sealed, meta, updated)
	default:
		return fmt.Errorf("update: %w", err)
	}
}

// resolveConflict интерактивно спрашивает пользователя как разрулить конфликт:
// keep local (наш новый вариант) → retry с актуальной revision;
// take remote (принять серверный) → отменить локальный edit;
// cancel → выход без изменений.
func resolveConflict(
	client *api.Client, key []byte,
	id, secretType, oldMeta string,
	localSealed []byte, localMeta string,
	remote *api.Secret,
) error {
	// Расшифровываем remote-вариант чтобы показать пользователю.
	remotePlain, err := crypto.Decrypt(key, remote.Data)
	if err != nil {
		return fmt.Errorf("decrypt remote (wrong master password?): %w", err)
	}
	// Тот же ключ применим и к localSealed — он свежесозданный.
	localPlain, err := crypto.Decrypt(key, localSealed)
	if err != nil {
		return fmt.Errorf("decrypt local: %w", err)
	}

	fmt.Println("═══ CONFLICT DETECTED ═══")
	fmt.Printf("Someone updated this secret while you were editing.\n\n")
	fmt.Printf("── REMOTE (revision %d, updated %s) ──\n", remote.Revision, remote.UpdatedAt.Format("2006-01-02 15:04:05"))
	printPayload(secretType, remotePlain, remote.Meta)
	fmt.Printf("\n── LOCAL (your edit) ──\n")
	printPayload(secretType, localPlain, localMeta)
	fmt.Println()

	fmt.Print("Keep [l]ocal (retry with remote's revision), take [r]emote, or [c]ancel? ")
	choice, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "l", "local":
		// Retry с revision от remote.
		updated, err := client.UpdateSecret(id, localSealed, localMeta, remote.Revision)
		if err != nil {
			return fmt.Errorf("retry update: %w", err)
		}
		fmt.Printf("kept local; new revision %d\n", updated.Revision)
	case "r", "remote":
		fmt.Println("took remote; local edit discarded")
	default:
		fmt.Println("cancelled; nothing changed")
	}
	return nil
}

// printPayload расшифровывает и печатает payload по типу секрета — для показа
// в conflict resolver.
func printPayload(secretType string, plaintext []byte, meta string) {
	fmt.Printf("meta: %s\n", meta)
	switch secretType {
	case typeLogin:
		var p loginPayload
		_ = json.Unmarshal(plaintext, &p)
		fmt.Printf("login: %s\npassword: %s\n", p.Login, p.Password)
	case typeText:
		var p textPayload
		_ = json.Unmarshal(plaintext, &p)
		fmt.Println(p.Content)
	case typeCard:
		var p cardPayload
		_ = json.Unmarshal(plaintext, &p)
		fmt.Printf("card %s exp %s holder %s cvv %s\n", p.Number, p.Expires, p.Holder, p.CVV)
	case typeBinary:
		fmt.Printf("(binary, %d bytes)\n", len(plaintext))
	}
}

func init() {
	// Флаги — те же переменные что и у put (putLoginLogin, putTextContent и т.д.)
	// уже определены в put.go, только регистрируем на updateCmd:
	updateCmd.PersistentFlags().StringVarP(&putMeta, "meta", "m", "", "новая метаинформация")

	updateLoginCmd.Flags().StringVar(&putLoginLogin, "login", "", "логин")
	updateLoginCmd.Flags().StringVar(&putLoginPass, "password", "", "пароль")

	updateTextCmd.Flags().StringVar(&putTextContent, "content", "", "текстовое содержимое")

	updateBinaryCmd.Flags().StringVar(&putBinaryFile, "file", "", "путь к файлу")

	updateCardCmd.Flags().StringVar(&putCardNumber, "number", "", "номер карты")
	updateCardCmd.Flags().StringVar(&putCardExpires, "expires", "", "срок MM/YY")
	updateCardCmd.Flags().StringVar(&putCardHolder, "holder", "", "имя держателя")
	updateCardCmd.Flags().StringVar(&putCardCVV, "cvv", "", "CVV")

	updateCmd.AddCommand(updateLoginCmd, updateTextCmd, updateBinaryCmd, updateCardCmd)
}
