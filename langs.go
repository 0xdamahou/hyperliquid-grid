package main

import "net/http"

func getLangTrans(r *http.Request) map[string]string {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}

	trans, ok := translations[lang]
	if !ok {
		trans = translations["en"]
	}
	return trans
}

var translations = map[string]map[string]string{
	"en": {
		"title":           "Grid Trading Profit/Loss",
		"summary":         "Trading Summary",
		"open_price":      "Open Price",
		"open_amount":     "Quantity",
		"grid_status":     "Grid Status",
		"match_count":     "Match Count",
		"symbol":          "Trading Pair",
		"trade_details":   "Trade Details",
		"matched_grid":    "Matched Grids",
		"unmatched_grids": "Unmatched Grids",
		"buy_time":        "Buy Time",
		"sell_time":       "Sell Time",
		"buy_level":       "Buy Level",
		"sell_level":      "Sell Level",
		"buy_price":       "Buy Price",
		"sell_price":      "Sell Price",
		"buy_amount":      "Buy Amount",
		"sell_amount":     "Sell Amount",
		"profit":          "Profit",
		"fee":             "Fee",
		"level":           "Level",
		"price":           "Price",
		"time":            "Time",
		"size":            "Size",
		"side":            "Side",
		"status_running":  "Running",
		"status_closed":   "Closed",
		"grid_list_title": "Grid List",
		"start_time":      "Start Time",
		"close_price":     "Close Price",
		"success_count":   "Matched Count",
		"unmatched_count": "Unmatched Count",
	},
	"zh": {
		"title":           "网格交易盈亏",
		"summary":         "交易概况",
		"open_price":      "开仓价格",
		"open_amount":     "开仓数量",
		"grid_status":     "网格状态",
		"match_count":     "匹配数量",
		"symbol":          "交易对",
		"trade_details":   "交易明细",
		"matched_grid":    "成功配对网格",
		"unmatched_grids": "未成功配对网格",
		"buy_time":        "买入时间",
		"sell_time":       "卖出时间",
		"buy_level":       "买入网格",
		"sell_level":      "卖出网格",
		"buy_price":       "买入价格",
		"sell_price":      "卖出价格",
		"buy_amount":      "买入数量",
		"sell_amount":     "卖出数量",
		"profit":          "利润",
		"fee":             "手续费",
		"level":           "网格层级",
		"price":           "价格",
		"time":            "时间",
		"size":            "头寸大小",
		"side":            "头寸方向",
		"status_running":  "运行",
		"status_closed":   "关闭",
		"grid_list_title": "网格列表",
		"start_time":      "开始时间",
		"close_price":     "关闭价格",
		"success_count":   "匹配次数",
		"unmatched_count": "未匹配次数",
	},
}
