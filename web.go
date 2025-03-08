package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

type TradePair struct {
	BuyLevel  int
	SellLevel int
	BuyTime   time.Time
	SellTime  time.Time
	BuyPrice  float64
	SellPrice float64
	BuySize   float64
	SellSize  float64
	Profit    float64
	Fee       float64
}

type GridData struct {
	GridID         int64
	OpenPrice      float64
	OpenNum        float64
	Status         string
	Symbol         string
	TradePairs     []TradePair
	UnMatched      []Orders
	Translations   map[string]string
	StartTime      time.Time
	ClosePrice     float64
	UnmatchedCount int
	MatchedCount   int
	Profit         float64
	Fee            float64
}

type GridDetail struct {
	GridID     int64
	Symbol     string
	StartTime  time.Time
	StartPrice float64
	StartSize  float64
	ClosePrice float64
	Status     string
	TradePairs []TradePair
	UnMatched  []Orders
	MatchCount int
	GridProfit float64
	Fee        float64

	Translations map[string]string
}

var storage *Storage

var WebPassword string

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil { // 只取 IPv4
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no valid IP found")
}
func generateRandomString(length int) string {
	// 定义字符集，可以根据需要调整
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())

	// 创建一个字节切片来存储随机字符
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		// 从字符集中随机选择一个字符
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func startWeb(port, key string) {
	if key == "" || key == "hello" {
		key = generateRandomString(8)
	}
	WebPassword = key
	http.HandleFunc("/", gridListHandler)
	http.HandleFunc("/grid/", gridHandler)
	http.HandleFunc("/grids", gridListHandler)
	ip, e := getLocalIP()
	if e != nil {
		log.Println(e)
	}
	if port == "" {
		port = ":8080"
	}
	prompt := ""
	if port == ":80" {
		prompt = fmt.Sprintf("http://%s?key=%s", ip, key)
	} else {
		prompt = fmt.Sprintf("http://%s%s?key=%s", ip, port, key)
	}
	fmt.Println("--------------------------------------------")
	fmt.Println("Please use the URL below to check the grid profitability status.")
	fmt.Println(prompt)
	fmt.Println("--------------------------------------------")
	http.ListenAndServe(port, nil)
}

func CheckPassword(w http.ResponseWriter, r *http.Request) bool {
	key := r.URL.Query().Get("key")
	if key != WebPassword {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func gridHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckPassword(w, r) {
		return
	}
	gridIDStr := r.URL.Path[len("/grid/"):]
	if gridIDStr == "" {
		http.Error(w, "Missing grid_id", http.StatusBadRequest)
		return
	}

	gridID, err := strconv.Atoi(gridIDStr)
	if err != nil {
		http.Error(w, "Invalid grid_id", http.StatusBadRequest)
		return
	}

	trans := getLangTrans(r)
	grid := storage.GetGridByGridId(int64(gridID))
	matched := make([]TradePair, 0)
	unMatched := make([]Orders, 0)
	totalProfit := 0.0
	totalFee := 0.0
	if grid.Level >= 0 {
		//如果当前level大于0,那么说明，卖出的要比买入的多，所以从卖出的最大值开始排序
		sellsOrders, e := storage.CompletedOrders(grid.ID, SellSide, true)
		if e != nil {
			log.Println(e)
		}
		buyOrders, e := storage.CompletedOrders(grid.ID, BuySide, true)
		if e != nil {
			log.Println(e)
		}

		for _, sellOrder := range sellsOrders {
			if len(buyOrders) > 0 {
				for _, buyOrder := range buyOrders {
					if sellOrder.Level-1 == buyOrder.Level {
						fee := LimitPriceOrderFee * (buyOrder.Price + sellOrder.Price) * buyOrder.Quantity
						profit := (sellOrder.Price-buyOrder.Price)*buyOrder.Quantity - fee
						pair := TradePair{
							BuyLevel:  buyOrder.Level,
							SellLevel: sellOrder.Level,
							BuyTime:   buyOrder.Time,
							SellTime:  sellOrder.Time,
							BuyPrice:  buyOrder.Price,
							SellPrice: sellOrder.Price,
							BuySize:   buyOrder.Quantity,
							SellSize:  sellOrder.Quantity,
							Profit:    profit,
							Fee:       fee,
						}
						totalProfit += profit
						totalFee += fee
						matched = append(matched, pair)
						buyOrders = buyOrders[1:]
						break
					} else if sellOrder.Level-1 > buyOrder.Level {
						unMatched = append(unMatched, sellOrder)
						totalFee += sellOrder.Fee
					} else if sellOrder.Level-1 < buyOrder.Level {
						unMatched = append(unMatched, buyOrder)
						buyOrders = buyOrders[1:]
						totalFee += buyOrder.Fee
						break
					}
				}

			} else {
				unMatched = append(unMatched, sellOrder)
				totalFee += sellOrder.Fee
			}

		}
	} else {
		//level小于0,说明买单多，需要从level底部开始匹配
		sellsOrders, e := storage.CompletedOrders(grid.ID, SellSide, false)
		if e != nil {
			log.Println(e)
		}
		buyOrders, e := storage.CompletedOrders(grid.ID, BuySide, false)
		if e != nil {
			log.Println(e)
		}
		for _, buyOrder := range buyOrders {
			if len(sellsOrders) > 0 {
				for _, sellOrder := range sellsOrders {

					if buyOrder.Level == sellOrder.Level-1 {

						fee := LimitPriceOrderFee * (buyOrder.Price + sellOrder.Price) * buyOrder.Quantity
						profit := (sellOrder.Price-buyOrder.Price)*buyOrder.Quantity - fee
						pair := TradePair{
							BuyLevel:  buyOrder.Level,
							SellLevel: sellOrder.Level,
							BuyTime:   buyOrder.Time,
							SellTime:  sellOrder.Time,
							BuyPrice:  buyOrder.Price,
							SellPrice: sellOrder.Price,
							BuySize:   buyOrder.Quantity,
							SellSize:  sellOrder.Quantity,
							Profit:    profit,
							Fee:       fee,
						}
						totalProfit += profit
						totalFee += fee
						matched = append(matched, pair)
						sellsOrders = sellsOrders[1:]
						break
					} else if buyOrder.Level > sellOrder.Level-1 {
						unMatched = append(unMatched, sellOrder)
						sellsOrders = sellsOrders[1:]
						totalFee += sellOrder.Fee
					} else if buyOrder.Level < sellOrder.Level-1 {
						unMatched = append(unMatched, buyOrder)
						totalFee += buyOrder.Fee
						break

					}
				}

			} else {
				unMatched = append(unMatched, buyOrder)
				totalFee += buyOrder.Fee
			}

		}
	}
	detail := GridDetail{GridID: grid.ID, Symbol: grid.Symbol, StartTime: grid.StartTime, StartPrice: grid.StartPrice, StartSize: grid.OpenQuantity, ClosePrice: nullFloat64(grid.EndPrice), Status: grid.Status, MatchCount: len(matched), TradePairs: matched, UnMatched: unMatched, GridProfit: totalProfit, Fee: totalFee, Translations: trans}

	tmpl, err := template.ParseFiles("./templates/grid_details.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, detail)
}
