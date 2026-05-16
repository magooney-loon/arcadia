package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// walletEdgesCollection — Layer 8: wallet graph (nodes + edges for 3D viz).
func walletEdgesCollection(app core.App) error {
	if collectionExists(app, "wallet_edges") {
		return nil
	}
	c := core.NewBaseCollection("wallet_edges")
	c.Fields.Add(&core.TextField{Name: "from_wallet", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_wallet", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "total_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.NumberField{Name: "last_seen_block"})
	c.Fields.Add(&core.BoolField{Name: "from_is_agent"})
	c.Fields.Add(&core.BoolField{Name: "to_is_agent"})
	c.AddIndex("idx_edges_unique", true, "from_wallet, to_wallet", "")
	c.AddIndex("idx_edges_from", false, "from_wallet", "")
	c.AddIndex("idx_edges_to", false, "to_wallet", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created wallet_edges collection")
	return nil
}
