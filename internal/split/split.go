package split

import (
	"math"

	"github.com/akksell/rbn/internal/store"
)

// Split divides totalAmount equally among roommates and returns one Debt per roommate.
func Split(totalAmount float64, roommates []store.Roommate) []store.Debt {
	if len(roommates) == 0 {
		return nil
	}
	share := totalAmount / float64(len(roommates))
	// Round to 2 decimal places to avoid floating point noise
	share = math.Round(share*100) / 100

	debts := make([]store.Debt, len(roommates))
	for i, r := range roommates {
		debts[i] = store.Debt{
			RoommateID: r.ID,
			Amount:     share,
			Status:     store.DebtStatusPending,
		}
	}

	// Adjust first debt for rounding so total matches
	sum := share * float64(len(roommates))
	diff := totalAmount - sum
	if diff != 0 {
		debts[0].Amount = math.Round((debts[0].Amount+diff)*100) / 100
	}

	return debts
}
