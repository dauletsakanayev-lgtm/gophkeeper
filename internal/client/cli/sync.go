package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/api"
	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/storage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Синхронизация локального кэша с сервером",
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Забрать все секреты с сервера в локальный кэш",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustSession()
		if err != nil {
			return err
		}
		db, err := storage.Open(storage.DefaultDBPath())
		if err != nil {
			return fmt.Errorf("open local db: %w", err)
		}
		defer db.Close()

		items, err := authClient(s).ListSecrets()
		if err != nil {
			return fmt.Errorf("fetch from server: %w", err)
		}
		cache := storage.NewCache(db)
		ctx := context.Background()
		for _, r := range items {
			if err := cache.Upsert(ctx, storage.LocalSecret{
				ID:        r.ID,
				Type:      r.Type,
				Data:      r.Data,
				Meta:      r.Meta,
				Revision:  r.Revision,
				CreatedAt: r.CreatedAt,
				UpdatedAt: r.UpdatedAt,
				Dirty:     false,
			}); err != nil {
				return fmt.Errorf("cache upsert %s: %w", r.ID, err)
			}
		}
		_ = storage.SetMeta(ctx, db, storage.MetaLogin, s.Login)
		_ = storage.SetMeta(ctx, db, storage.MetaLastSyncAt, time.Now().Format(time.RFC3339))

		fmt.Printf("pulled %d secret(s) to %s\n", len(items), storage.DefaultDBPath())
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Отправить локальные dirty-секреты на сервер",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := mustSession()
		if err != nil {
			return err
		}
		db, err := storage.Open(storage.DefaultDBPath())
		if err != nil {
			return fmt.Errorf("open local db: %w", err)
		}
		defer db.Close()
		cache := storage.NewCache(db)
		client := authClient(s)

		ctx := context.Background()
		dirty, err := cache.ListDirty(ctx)
		if err != nil {
			return fmt.Errorf("list dirty: %w", err)
		}
		if len(dirty) == 0 {
			fmt.Println("nothing to push")
			return nil
		}

		pushed, conflicts, failed := 0, 0, 0
		for _, item := range dirty {
			// Отправляем как UPDATE (пробуем PUT). Если сервер не знает такой id — Create.
			_, err := client.UpdateSecret(item.ID, item.Data, item.Meta, item.Revision)
			switch {
			case err == nil:
				_ = cache.ClearDirty(ctx, item.ID)
				pushed++
			case errors.Is(err, api.ErrConflict):
				// Другой клиент опередил — оставляем dirty, пусть пользователь разрулит через `update`.
				conflicts++
				fmt.Printf("conflict on %s — resolve with 'gophkeeper update <type> %s ...'\n", item.ID, item.ID)
			case errors.Is(err, api.ErrNotFound):
				// На сервере нет — создаём заново.
				if _, err := client.CreateSecret(item.Type, item.Data, item.Meta); err != nil {
					failed++
					fmt.Printf("failed to create %s: %v\n", item.ID, err)
				} else {
					_ = cache.ClearDirty(ctx, item.ID)
					pushed++
				}
			default:
				failed++
				fmt.Printf("failed %s: %v\n", item.ID, err)
			}
		}
		fmt.Printf("pushed=%d conflicts=%d failed=%d\n", pushed, conflicts, failed)
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncPullCmd, syncPushCmd)
}
