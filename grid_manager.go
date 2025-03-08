// grid_manager.go
package main

import (
	"fmt"
	hp "github.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type GridManager struct {
	Symbol      string
	IsRunning   bool
	Storage     *Storage
	Hyper       *hp.Hyperliquid
	GridStep    float64
	CenterPrice float64
	Having      float64
	Each        float64
	Level       int
	Precision   int
	UpId        int64
	DownId      int64
	GridId      int64
	mu          sync.Mutex
	havingMux   sync.Mutex
	stopChan    chan struct{}
}

func (gm *GridManager) SetHaving(having float64) {
	gm.havingMux.Lock()
	defer gm.havingMux.Unlock()
	gm.Having = having
}

func (gm *GridManager) GetHaving() float64 {
	gm.havingMux.Lock()
	defer gm.havingMux.Unlock()
	return gm.Having
}

func NewGridManager(symbol string, storage *Storage, hpl *hp.Hyperliquid, each float64, gridStep float64, precious int, level int) *GridManager {
	log.Printf("%s %f %f %d\n", symbol, each, gridStep, precious)
	return &GridManager{
		Symbol:    symbol,
		IsRunning: false,
		Storage:   storage,
		Hyper:     hpl,
		Each:      each,
		Level:     level,
		GridStep:  gridStep,
		Precision: precious,
		stopChan:  make(chan struct{}),
	}
}

func (gm *GridManager) StartGrid(quantity float64) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	runningGrid, _ := gm.Storage.GetRunningGridBySymbol(gm.Symbol)

	if runningGrid != nil {
		if quantity*runningGrid.OpenQuantity > 0 {
			log.Printf("Grid %s is already running\n", gm.Symbol)
			_, _ = gm.Hyper.CancelAllOrdersByCoin(gm.Symbol)
			gm.Level = runningGrid.Level
			gm.SetHaving(runningGrid.OpenQuantity)
			gm.CenterPrice = runningGrid.StartPrice
			gm.GridId = runningGrid.ID
			gm.UpdateLevel(gm.Level)
			gm.stopChan = make(chan struct{})
			gm.IsRunning = true
			go gm.monitorOrders()
			return nil
		} else {
			//Different side grid
			err := gm.StopGrid()
			if err != nil {
				return fmt.Errorf("failed to stop existing grid: %v", err)
			}
			gm.Level = 0
		}

	}
	avgPrice, size, oid := gm.MarketOrder(quantity)
	grid := NewGrid(gm.Symbol, avgPrice, quantity, oid, 0)
	//gm.Having = size
	gm.SetHaving(size)
	gm.CenterPrice = avgPrice
	gridID, err := gm.Storage.InsertGrid(grid)

	if err != nil {
		return fmt.Errorf("failed to insert grid: %v", err)
	}
	gm.GridId = gridID
	//Put Grid orders
	gm.UpdateLevel(gm.Level)
	gm.stopChan = make(chan struct{})
	gm.IsRunning = true
	go gm.monitorOrders()

	return nil
}

func (gm *GridManager) StopGrid() error {

	if !gm.IsRunning {
		return fmt.Errorf("no running grid to stop for %s", gm.Symbol)
	}

	grid, _ := gm.Storage.GetRunningGridBySymbol(gm.Symbol)

	if grid == nil {
		return fmt.Errorf("no running grid found for %s", gm.Symbol)
	}

	_, _ = gm.Hyper.CancelAllOrdersByCoin(gm.Symbol)

	//ClosePosition：&{Status:ok Response:{Type:order Data:{Statuses:[{Resting:{OrderId:0 Cloid:} Filled:{OrderId:75503717647 AvgPx:140.37 TotalSz:0.1 Cloid:} Error: Status:}]}}}

	closeRes, err := gm.Hyper.ClosePosition(gm.Symbol)
	if err != nil {
		log.Printf("failed to close position: %v\n", err)
	}

	closePrice := closeRes.Response.Data.Statuses[0].Filled.AvgPx
	closeQuantity := closeRes.Response.Data.Statuses[0].Filled.TotalSz
	closeId := int64(closeRes.Response.Data.Statuses[0].Filled.OrderId)

	grid.EndTime = timePtr(time.Now())
	grid.EndPrice = float64Ptr(closePrice)
	grid.CloseQuantity = float64Ptr(closeQuantity)
	grid.CloseAmount = float64Ptr(closePrice * closeQuantity)
	grid.CloseOrderID = int64Ptr(closeId)

	err = gm.Storage.UpdateGrid(grid.ID, *grid)

	profit, fee, err := gm.calculateProfitAndFee(grid.ID)
	if err != nil {
		log.Printf("failed to calculate profit and fee: %v\n", err)
	}

	grid.Profit = float64Ptr(profit)
	grid.Fee = float64Ptr(fee)
	grid.Status = TerminatedStatus

	err = gm.Storage.UpdateGrid(grid.ID, *grid)
	if err != nil {
		log.Printf("failed to update grid: %v\n", err)
	}

	gm.IsRunning = false
	close(gm.stopChan)

	return nil
}
func (gm *GridManager) LevelPrice(level int) float64 {

	price := gm.CenterPrice
	switch {
	case level > 0:
		//log.Printf("Level %d %f,%f,%f,%f\n", level, 1+gm.GridStep, float64(level), math.Pow(1+gm.GridStep, float64(level)), gm.CenterPrice*math.Pow(1+gm.GridStep, float64(level)))
		price = gm.CenterPrice * math.Pow(1+gm.GridStep, float64(level))
		break
	case level < 0:
		//log.Printf("Level %d %f,%f,%f,%f\n", level, 1-gm.GridStep, float64(level), math.Pow(1-gm.GridStep, float64(-level)), gm.CenterPrice*math.Pow(1-gm.GridStep, float64(-level)))
		price = gm.CenterPrice * math.Pow(1-gm.GridStep, float64(-level))
		break
	default:
		price = gm.CenterPrice
	}
	price = FormatFloat2(price, gm.Precision)
	//log.Printf("Level price for %s %d %f %f\n", gm.Symbol, level, gm.CenterPrice, price)
	return price
}
func (gm *GridManager) UpdateLevel(level int) {
	gm.Level = level
	upPrice := gm.LevelPrice(level + 1)
	downPrice := gm.LevelPrice(level - 1)
	having := gm.GetHaving()
	upReduceOnly := having > gm.Each
	downReduceOnly := having < -1*gm.Each
	log.Printf("%s Level %d Up %.4f %v Low: %.4f %v \n", gm.Symbol, level, upPrice, upReduceOnly, downPrice, downReduceOnly)
	gm.UpId = gm.LimitOrder(-1*gm.Each, upPrice, upReduceOnly)
	gm.DownId = gm.LimitOrder(gm.Each, downPrice, downReduceOnly)

	err := gm.Storage.UpdateGridLevel(gm.GridId, level)
	if err != nil {
		log.Printf("Warning: failed to update level: %v", err)
		return
	}
	_, err = gm.Storage.InsertOrder(Orders{
		GridID:   gm.GridId,
		Level:    level + 1,
		Price:    upPrice,
		Quantity: gm.Each,
		Amount:   upPrice * gm.Each,
		Time:     time.Now(),
		Side:     SellSide,
		Fee:      upPrice * gm.Each * LimitPriceOrderFee,
		OrderID:  gm.UpId,
		Status:   InitStatus,
	})
	if err != nil {
		log.Printf("Warning: failed to insert grid order: %v", err)
		return
	}
	_, _ = gm.Storage.InsertOrder(Orders{
		GridID:   gm.GridId,
		Level:    level - 1,
		Price:    downPrice,
		Quantity: gm.Each,
		Amount:   downPrice * gm.Each,
		Time:     time.Now(),
		Side:     BuySide,
		Fee:      downPrice * gm.Each * LimitPriceOrderFee,
		OrderID:  gm.DownId,
		Status:   InitStatus,
	})

}
func FormatFloat2(num float64, decimal int) float64 {
	factor := math.Pow(10, float64(decimal))
	// 四舍五入到指定的小数位数
	return math.Round(num*factor) / factor

}

func (gm *GridManager) LimitOrder(quantity float64, price float64, reduceOnly bool) int64 {

	//&{Status:ok Response:{Type:order Data:{Statuses:[{Resting:{OrderId:75669481191 Cloid:} Filled:{OrderId:0 AvgPx:0 TotalSz:0 Cloid:} Error: Status:}]}}}
	//log.Printf("%s %f %f %+v", gm.Symbol, quantity, price, reduceOnly)
	orderResp, err := gm.Hyper.LimitOrder(hp.TifGtc, gm.Symbol, quantity, price, reduceOnly)
	if err != nil {
		log.Printf("Failed to place limit order: %v", err)
		return 0
	}
	if orderResp.Status != OkStatus {
		log.Printf("Failed to place limit order: %v", orderResp)
		return 0
	}
	id := int64(orderResp.Response.Data.Statuses[0].Resting.OrderId)
	if id == 0 {
		id = int64(orderResp.Response.Data.Statuses[0].Filled.OrderId)
	}
	return id
}

func (gm *GridManager) MarketOrder(quantity float64) (avgPrice float64, size float64, oid int64) {
	slippage := 0.5
	// marketOrder: &{Status:ok Response:{Type:order Data:{Statuses:[{Resting:{OrderId:0 Cloid:} Filled:{OrderId:75501601271 AvgPx:140.39 TotalSz:0.1 Cloid:} Error: Status:}]}}}
	orderResp, err := gm.Hyper.MarketOrder(gm.Symbol, quantity, &slippage)

	if err != nil {
		log.Printf("failed to place initial order: %v", err)
		return 0, 0, 0
	}

	if orderResp.Status != OkStatus {
		log.Printf("failed to place initial order: %v", orderResp)
		return 0, 0, 0
	}
	avgPrice = orderResp.Response.Data.Statuses[0].Filled.AvgPx
	size = orderResp.Response.Data.Statuses[0].Filled.TotalSz
	oid = int64(orderResp.Response.Data.Statuses[0].Filled.OrderId)
	return
}

func (gm *GridManager) monitorOrders() {
	sigChan := make(chan os.Signal, 1)

	// Subscription SIGTERM and SIGINT (Ctrl+C)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-gm.stopChan:
			return
		case <-sigChan:

			log.Printf("Close orders and try to exit")
			_, _ = gm.Hyper.CancelAllOrdersByCoin(gm.Symbol)
			_ = gm.Storage.UpdateOrderStatus(gm.UpId, InitStatus, CancelledStatus)
			_ = gm.Storage.UpdateOrderStatus(gm.DownId, InitStatus, CancelledStatus)
		case <-ticker.C:
			gm.mu.Lock()
			if !gm.IsRunning {
				gm.mu.Unlock()
				return
			}
			result, e := IsOrderFilled(gm.Hyper, gm.Symbol, []int64{gm.UpId, gm.DownId})
			if e != nil {
				log.Printf("Failed to check order status: %v", e)
				gm.mu.Unlock()
				continue
			}
			if downDealt, _ := result[gm.DownId]; downDealt {
				log.Printf("%s 向下成交(%.4f)\n", gm.Symbol, gm.LevelPrice(gm.Level-1))
				//向下成交，取消上面的订单，数据库中，将上面的订单标志为取消，将下方的订单标识为完成，并下移
				_, err := gm.Hyper.CancelOrderByOID(gm.Symbol, gm.UpId)
				if err != nil {
					log.Printf("Failed to cancel order: %v", err)
				}
				err = gm.Storage.UpdateOrderStatus(gm.UpId, InitStatus, CancelledStatus)
				if err != nil {
					log.Printf("Failed to update order status: %v", err)
				}

				err = gm.Storage.UpdateOrderStatus(gm.DownId, InitStatus, CompletedStatus)
				if err != nil {
					log.Printf("Failed to update order status: %v", err)
				}
				gm.UpdateLevel(gm.Level - 1)

			}
			if upDealt, _ := result[gm.UpId]; upDealt {
				log.Printf("%s 向上成交(%.4f)\n", gm.Symbol, gm.LevelPrice(gm.Level+1))

				_, err := gm.Hyper.CancelOrderByOID(gm.Symbol, gm.DownId)
				if err != nil {
					log.Printf("Failed to cancel order: %v", err)
				}
				err = gm.Storage.UpdateOrderStatus(gm.DownId, InitStatus, CancelledStatus)
				if err != nil {
					log.Printf("Failed to update order status: %v", err)
				}
				err = gm.Storage.UpdateOrderStatus(gm.UpId, InitStatus, CompletedStatus)
				if err != nil {
					log.Printf("Failed to update order status: %v", err)
				}

				gm.UpdateLevel(gm.Level + 1)

			}

			gm.mu.Unlock()
		}
	}
}

func (gm *GridManager) calculateProfitAndFee(gridID int64) (float64, float64, error) {
	grid := gm.Storage.GetGridByGridId(gridID)
	marketFees := MarketOrderFee * (math.Abs(grid.OpenAmount) + math.Abs(nullFloat64(grid.CloseQuantity)*nullFloat64(grid.EndPrice)))

	orders, err := gm.Storage.GetOrdersByGridID(gridID)
	if err != nil {
		return 0, 0, err
	}

	var totalBuy, totalSell, gridFee float64
	for _, order := range orders {
		if order.Status == CompletedStatus {
			gridFee += order.Fee
			if order.Side == SellSide {
				totalSell += order.Amount
			} else {
				totalBuy += order.Amount
			}
		}
	}
	profit := 0.0
	if grid.OpenQuantity > 0 {
		profit = math.Abs(nullFloat64(grid.CloseQuantity)*nullFloat64(grid.EndPrice)) - math.Abs(grid.OpenQuantity*grid.StartPrice)
	} else {
		profit = math.Abs(grid.OpenQuantity*grid.StartPrice) - math.Abs(nullFloat64(grid.CloseQuantity)*nullFloat64(grid.EndPrice))
	}

	profit = totalSell - totalBuy - gridFee - marketFees
	return profit, gridFee + marketFees, nil
}

func int64Ptr(i int64) *int64        { return &i }
func intPtr(i int) *int              { return &i }
func float64Ptr(f float64) *float64  { return &f }
func timePtr(t time.Time) *time.Time { return &t }
func stringPtr(s string) *string     { return &s }
func nullFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
