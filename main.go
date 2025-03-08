package main

import (
	hp "github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
	"log"
	"sync"
	"time"
)

func UpdateLeverage(hpl *hp.Hyperliquid, coin string, leverage int) {
	_, err := hpl.UpdateLeverage(coin, true, leverage)
	if err != nil {
		log.Println(err)
		return
	}
}

// GridManagerPool manages multiple GridManager instances
type GridManagerPool struct {
	managers sync.Map // Using built-in sync.Map for concurrent access
}

var gridManagerPool = NewGridManagerPool()

// NewGridManagerPool creates a new grid manager pool
func NewGridManagerPool() *GridManagerPool {
	return &GridManagerPool{}
}

// AddManager adds a grid manager to the pool
func (pool *GridManagerPool) AddManager(name string, manager *GridManager) {
	pool.managers.Store(name, manager)
}

// GetManager retrieves a grid manager from the pool
func (pool *GridManagerPool) GetManager(name string) (*GridManager, bool) {
	manager, exists := pool.managers.Load(name)
	if !exists {
		return nil, false
	}
	return manager.(*GridManager), true
}

// DeleteManager removes a grid manager from the pool
func (pool *GridManagerPool) DeleteManager(name string) {
	pool.managers.Delete(name)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config, e := LoadConfig("./config.json")
	if e != nil {
		log.Fatal(e)
		return
	}
	storage, e = NewStorage(config.DBConnection)
	if e != nil {
		log.Fatal(e)
		return
	}
	if config.EnableWeb {
		go startWeb(config.WebPort, config.WebPassword)
	}

	gridsConfig, e := LoadGridConfigs("./grid_config.json")
	if e != nil {
		log.Fatal(e)
		return
	}

	hpl := hp.NewHyperliquid(&hp.HyperliquidClientConfig{IsMainnet: true, AccountAddress: config.APIKey, PrivateKey: config.APISecret})
	for _, grid := range gridsConfig {
		UpdateLeverage(hpl, grid.Symbol, grid.Leverage)
		if grid.Enable {
			runningGrid, _ := storage.GetRunningGridBySymbol(grid.Symbol)
			level := 0
			if runningGrid != nil {
				level = runningGrid.Level
			}
			gm := NewGridManager(grid.Symbol, storage, hpl, grid.GridSize, grid.GridStep, grid.PricePrecision, level)
			gridManagerPool.AddManager(grid.Symbol, gm)
		}
	}

	for _, grid := range gridsConfig {
		if grid.Enable {
			pm, _ := gridManagerPool.GetManager(grid.Symbol)
			switch grid.Act {
			case StartAct:
				err := pm.StartGrid(grid.InitialSize)
				if err != nil {
					log.Println(err)
				}
			case StopAct:
				err := pm.StopGrid()
				if err != nil {
					log.Println(err)
				}
			}

		}
	}

	for {
		accountState, err := hpl.InfoAPI.GetUserState(hpl.AccountAddress())
		if err != nil {
			log.Println(err)
		}
		for _, position := range accountState.AssetPositions {
			//log.Printf("%s %+v", position.Position.Coin, position.Position.Szi)
			gm, ok := gridManagerPool.GetManager(position.Position.Coin)
			if ok {
				gm.SetHaving(position.Position.Szi)
			}
		}
		time.Sleep(3 * time.Minute)
	}
}
