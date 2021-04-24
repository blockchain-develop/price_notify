package test

import (
	"fmt"
	"os"
	"price_notify/basedef"
	"price_notify/coinpricedao"
	"price_notify/coinpricelisten"
	"price_notify/conf"
	"testing"
)

func TestListenCoinPrice(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("current directory: %s\n", dir)
	config := conf.NewConfig("./../../conf/config_mainnet.json")
	if config == nil {
		panic("read config failed!")
	}
	dao := coinpricedao.NewCoinPriceDao(basedef.SERVER_PRICE, config.DBConfig)
	if dao == nil {
		panic("server is not valid")
	}
	priceListenConfig := config.CoinPriceListenConfig
	priceMarkets := make([]coinpricelisten.PriceMarket, 0)
	for _, cfg := range priceListenConfig {
		priceMarket := coinpricelisten.NewPriceMarket(cfg)
		priceMarkets = append(priceMarkets, priceMarket)
	}
	cpListen := coinpricelisten.NewCoinPriceListen(config.CoinPriceUpdateSlot, priceMarkets, dao)
	cpListen.ListenPrice()
}


