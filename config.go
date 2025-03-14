// config.go
package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	APIKey       string `json:"api_key"`
	APISecret    string `json:"api_secret"`
	EnableWeb    bool   `json:"enable_web"`
	WebPassword  string `json:"web_password"`
	WebPort      string `json:"web_port"`
	DBConnection string `json:"db_connection"`
}

// GridConfig 网格交易配置（针对每个交易对）
type GridConfig struct {
	Symbol         string  `json:"symbol"`       // Coin symbol
	InitialSize    float64 `json:"initial_size"` // 初始订单数量
	GridStep       float64 `json:"grid_step"`    // 网格步长（如 0.5%）
	GridSize       float64 `json:"grid_size"`    // 每次网格交易数量
	Leverage       int     `json:"leverage"`     // 杠杆倍数
	PricePrecision int     `json:"price_precision"`
	Act            string  `json:"act"`    //Act:Start,Stop
	Enable         bool    `json:"enable"` // 是否启用该网格
}

// LoadConfig 从 JSON 文件加载全局配置
func LoadConfig(filePath string) (Config, error) {
	var cfg Config
	file, err := os.Open(filePath)
	if err != nil {
		return cfg, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadGridConfigs 从 JSON 文件加载网格交易配置数组
func LoadGridConfigs(filePath string) ([]GridConfig, error) {
	var gridConfigs []GridConfig
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&gridConfigs)
	if err != nil {
		return nil, err
	}
	return gridConfigs, nil
}
