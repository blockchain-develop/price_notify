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

package conf

import (
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"price_notify/basedef"
)

type DBConfig struct {
	URL      string
	User     string
	Password string
	Scheme   string
	Debug    bool
}

type Restful struct {
	Url string
	Key string
}

type CoinPriceListenConfig struct {
	MarketName string
	Nodes      []*Restful
}

type Config struct {
	Server string
	CoinPriceUpdateSlot   int64
	CoinPriceListenConfig []*CoinPriceListenConfig
	DBConfig              *DBConfig
}

func NewConfig(filePath string) *Config {
	fileContent, err := basedef.ReadFile(filePath)
	if err != nil {
		logs.Error("NewServiceConfig: failed, err: %s", err)
		return nil
	}
	config := &Config{}
	err = json.Unmarshal(fileContent, config)
	if err != nil {
		logs.Error("NewServiceConfig: failed, err: %s", err)
		return nil
	}
	return config
}
