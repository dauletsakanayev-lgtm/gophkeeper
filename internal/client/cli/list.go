package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/dauletsakanayev-lgtm/gophkeeper/internal/client/storage"
	"github.com/spf13/cobra"
)

var listLocal bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Показать список секретов (id / type / meta / updated)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listLocal {
			return listFromCache()
		}
		return listFromServer()
	},
}

func listFromServer() error {
	s, err := mustSession()
	if err != nil {
		return err
	}
	items, err := authClient(s).ListSecrets()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if len(items) == 0 {
		fmt.Println("no secrets")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tMETA\tUPDATED")
	for _, it := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			it.ID, it.Type, it.Meta, it.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	return w.Flush()
}

func listFromCache() error {
	db, err := storage.Open(storage.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("open local db: %w", err)
	}
	defer db.Close()
	items, err := storage.NewCache(db).List(context.Background())
	if err != nil {
		return fmt.Errorf("list local: %w", err)
	}
	if len(items) == 0 {
		fmt.Println("no local secrets (run 'sync pull' first)")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tMETA\tREV\tDIRTY\tUPDATED")
	for _, it := range items {
		dirty := ""
		if it.Dirty {
			dirty = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			it.ID, it.Type, it.Meta, it.Revision, dirty, it.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	_ = time.Now // silence unused if needed
	return w.Flush()
}

func init() {
	listCmd.Flags().BoolVar(&listLocal, "local", false, "читать из локального кэша (~/.gophkeeper/local.db)")
}
