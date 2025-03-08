package main

import (
	"html/template"
	"log"
	"net/http"
)

func gridListHandler(w http.ResponseWriter, r *http.Request) {
	if !CheckPassword(w, r) {
		return
	}
	trans := getLangTrans(r)

	//grids := getGridList()
	grids := storage.GetAllGrids()
	var gridsData []GridData

	for _, grid := range grids {
		sellsOrders, e := storage.CompletedOrders(grid.ID, SellSide, false)
		if e != nil {
			log.Println(e)
		}
		BuyOrders, e := storage.CompletedOrders(grid.ID, BuySide, false)
		if e != nil {
			log.Println(e)
		}
		matched := 0
		unmatchedCount := 0
		if len(sellsOrders) > len(BuyOrders) {
			matched = len(BuyOrders)
			unmatchedCount = len(sellsOrders) - len(BuyOrders)
		} else {
			matched = len(sellsOrders)
			unmatchedCount = len(BuyOrders) - len(sellsOrders)
		}

		g := GridData{GridID: grid.ID, OpenPrice: grid.StartPrice, OpenNum: grid.OpenQuantity, Status: grid.Status, Symbol: grid.Symbol, TradePairs: nil, Translations: nil, StartTime: grid.StartTime,
			ClosePrice: nullFloat64(grid.EndPrice), UnmatchedCount: unmatchedCount, MatchedCount: matched, Profit: nullFloat64(grid.Profit), Fee: nullFloat64(grid.Fee)}
		gridsData = append(gridsData, g)
	}
	data := struct {
		Grids        []GridData
		GridSize     int
		Key          string
		Translations map[string]string
	}{
		Grids:        gridsData,
		GridSize:     len(grids),
		Key:          WebPassword,
		Translations: trans,
	}

	tmpl, err := template.ParseFiles("./templates/grid_list.html")
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}
