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

package pricedao

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"price_notify/basedef"
	"price_notify/conf"
	"price_notify/models"
)

type PriceDao struct {
	dbCfg *conf.DBConfig
	db    *gorm.DB
}

func NewPriceDao(dbCfg *conf.DBConfig) *PriceDao {
	dao := &PriceDao{
		dbCfg: dbCfg,
	}
	Logger := logger.Default
	if dbCfg.Debug == true {
		Logger = Logger.LogMode(logger.Info)
	}
	db, err := gorm.Open(mysql.Open(dbCfg.User+":"+dbCfg.Password+"@tcp("+dbCfg.URL+")/"+
		dbCfg.Scheme+"?charset=utf8"), &gorm.Config{Logger: Logger})
	if err != nil {
		panic(err)
	}
	dao.db = db
	return dao
}

func (dao *PriceDao) SavePrices(tokens []*models.TokenBasic) error {
	if tokens != nil && len(tokens) > 0 {
		res := dao.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(tokens)
		if res.Error != nil {
			return res.Error
		}
	}
	return nil
}

func (dao *PriceDao) GetTokens() ([]*models.TokenBasic, error) {
	tokens := make([]*models.TokenBasic, 0)
	res := dao.db.Preload("PriceMarkets").Find(&tokens)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("no record!")
	}
	return tokens, nil
}

func (dao *PriceDao) AddTokens(tokens []*models.TokenBasic) error {
	if tokens != nil && len(tokens) > 0 {
		res := dao.db.Save(tokens)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("add tokens failed!")
		}
	}
	return nil
}

func (dao *PriceDao) Name() string {
	return basedef.SERVER_PRICE
}
