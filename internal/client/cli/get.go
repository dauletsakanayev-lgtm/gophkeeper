package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/crypto"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/storage"
	"github.com/spf13/cobra"
)

var (
	getOutput string
	getLocal  bool
)

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Получить и расшифровать секрет",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		var (
			secretType string
			data       []byte
			meta       string
			revision   int
		)

		if getLocal {
			db, err := storage.Open(storage.DefaultDBPath())
			if err != nil {
				return fmt.Errorf("open local db: %w", err)
			}
			defer db.Close()
			ls, err := storage.NewCache(db).Get(context.Background(), id)
			if err != nil {
				if errors.Is(err, storage.ErrLocalNotFound) {
					return fmt.Errorf("no such secret in local cache; try 'sync pull' or omit --local")
				}
				return fmt.Errorf("local get: %w", err)
			}
			// Для decrypt нужен только login (для деривации ключа) — берём из meta таблицы.
			secretType, data, meta, revision = ls.Type, ls.Data, ls.Meta, ls.Revision
		} else {
			s, err := mustSession()
			if err != nil {
				return err
			}
			secret, err := authClient(s).GetSecret(id)
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}
			secretType, data, meta, revision = secret.Type, secret.Data, secret.Meta, secret.Revision
		}

		// Логин нужен для деривации ключа. Из session (online) или из meta (offline).
		login, err := loginForKey()
		if err != nil {
			return err
		}
		key, err := deriveMasterKey(login)
		if err != nil {
			return err
		}
		plaintext, err := crypto.Decrypt(key, data)
		if err != nil {
			return fmt.Errorf("decrypt (wrong master password?): %w", err)
		}

		fmt.Printf("id:       %s\n", id)
		fmt.Printf("type:     %s\n", secretType)
		fmt.Printf("meta:     %s\n", meta)
		fmt.Printf("revision: %d\n", revision)
		fmt.Println("---")

		return printSecretByType(secretType, plaintext)
	},
}

// loginForKey возвращает login для деривации ключа. Онлайн — из session,
// оффлайн (--local без активной session) — из meta локальной БД.
func loginForKey() (string, error) {
	if s, err := mustSession(); err == nil {
		return s.Login, nil
	}
	// fallback: читаем login из meta local db
	db, err := storage.Open(storage.DefaultDBPath())
	if err != nil {
		return "", fmt.Errorf("open local db: %w", err)
	}
	defer db.Close()
	login, ok, err := storage.GetMeta(context.Background(), db, storage.MetaLogin)
	if err != nil {
		return "", err
	}
	if !ok || login == "" {
		return "", fmt.Errorf("login unknown; run 'gophkeeper login' or 'sync pull' online first")
	}
	return login, nil
}

// printSecretByType — вынесен, чтобы не дублировать в conflict resolver и get.
func printSecretByType(secretType string, plaintext []byte) error {
	switch secretType {
	case typeLogin:
		var p loginPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		fmt.Printf("login:    %s\npassword: %s\n", p.Login, p.Password)
	case typeText:
		var p textPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
		}
		fmt.Println(p.Content)
	case typeCard:
		var p cardPayload
		if err := json.Unmarshal(plaintext, &p); err != nil {
			return err
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
			_, _ = os.Stdout.Write(plaintext)
		}
	default:
		fmt.Printf("(unknown type, raw plaintext, %d bytes)\n", len(plaintext))
	}
	return nil
}

// silence unused import if needed — api.Secret использована выше через authClient.
var _ = api.ErrConflict

func init() {
	getCmd.Flags().StringVarP(&getOutput, "output", "o", "",
		"для binary: путь к файлу вывода (по умолчанию stdout)")
	getCmd.Flags().BoolVar(&getLocal, "local", false, "читать из локального кэша")
}
