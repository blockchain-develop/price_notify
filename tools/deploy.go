/*
 * Copyright (C) 2020 The poly network Authors
 * This file is part of The poly network library.
 *
 * The  poly network  is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The  poly network  is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with The poly network .  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"price_notify/coinpricedao"
	"price_notify/models"
	"price_notify/pricenotifydao"
)

func startUpdate(cfg *UpdateConfig) {
	dbCfg := cfg.DBConfig
	Logger := logger.Default
	if dbCfg.Debug == true {
		Logger = Logger.LogMode(logger.Info)
	}
	db, err := gorm.Open(mysql.Open(dbCfg.User+":"+dbCfg.Password+"@tcp("+dbCfg.URL+")/"+
		dbCfg.Scheme+"?charset=utf8"), &gorm.Config{Logger: Logger})
	if err != nil {
		panic(err)
	}
	err = db.Debug().AutoMigrate(&models.TokenBasic{}, &models.PriceMarket{}, &models.PriceNotify{})
	if err != nil {
		panic(err)
	}
	//
	{
		db.Where("1 = 1").Delete(&models.PriceMarket{})
		db.Where("1 = 1").Delete(&models.PriceNotify{})
		db.Where("1 = 1").Delete(&models.TokenBasic{})
	}
	{
		dao := coinpricedao.NewCoinPriceDao(cfg.Server, cfg.DBConfig)
		if dao == nil {
			panic("server is invalid")
		}
		dao.AddTokens(cfg.TokenBasics)
	}
	{
		dao := pricenotifydao.NewPriceNotifyDao(cfg.Server, cfg.DBConfig)
		if dao == nil {
			panic("server is invalid")
		}
		dao.AddNotifies(cfg.PriceNotifies)
	}
}
