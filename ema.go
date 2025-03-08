// hyper.go
package main

import (
	"fmt"
	hp "github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
)

func IsOrderFilled(hpl *hp.Hyperliquid, symbol string, orderIDs []int64) (map[int64]bool, error) {

	fills, err := hpl.GetAccountFills()
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %v", err)
	}

	result := make(map[int64]bool)
	for _, oid := range orderIDs {
		result[oid] = false
	}

	for _, fill := range *fills {
		for _, oid := range orderIDs {
			if int64(fill.Oid) == oid && fill.Coin == symbol {
				result[oid] = true // order filled
			}
		}
	}

	return result, nil
}
