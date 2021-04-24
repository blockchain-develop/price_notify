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

package models

type TokenBasic struct {
	Name         string         `gorm:"primaryKey;size:64;not null"`
	Price        int64          `gorm:"size:64;not null"`
	PriceInd          uint64         `gorm:"type:bigint(20);not null"`
	Time         int64          `gorm:"type:bigint(20);not null"`
	PriceMarkets []*PriceMarket `gorm:"foreignKey:TokenBasicName;references:Name"`
}

type PriceMarket struct {
	TokenBasicName string      `gorm:"primaryKey;size:64;not null"`
	MarketName     string      `gorm:"primaryKey;size:64;not null"`
	Name           string      `gorm:"size:64;not null"`
	Price          int64       `gorm:"type:bigint(20);not null"`
	PriceInd            uint64      `gorm:"type:bigint(20);not null"`
	Time           int64       `gorm:"type:bigint(20);not null"`
	TokenBasic     *TokenBasic `gorm:"foreignKey:TokenBasicName;references:Name"`
}

type PriceNotify struct {
	Id int64  `gorm:"primaryKey;autoIncrement"`
	Price int64          `gorm:"size:64;not null"`
	TokenBasicName string `gorm:"size:64;not null"`
	TokenBasic     *TokenBasic `gorm:"foreignKey:TokenBasicName;references:Name"`
}
