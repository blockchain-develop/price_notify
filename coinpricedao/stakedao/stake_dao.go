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

package stakedao

import (
	"encoding/json"
	"fmt"
	"price_notify/basedef"
	"price_notify/models"
)

type StakeDao struct {
	tokenBasics []*models.TokenBasic
}

func NewStakeDao() *StakeDao {
	stakeDao := &StakeDao{}
	return stakeDao
}

func (dao *StakeDao) SavePrices(tokens []*models.TokenBasic) error {
	{
		json, _ := json.Marshal(tokens)
		fmt.Printf("tokens: %s\n", json)
	}
	return nil
}

func (dao *StakeDao) GetTokens() ([]*models.TokenBasic, error) {
	return dao.tokenBasics, nil
}

func (dao *StakeDao) AddTokens(tokens []*models.TokenBasic) error {
	return nil
}

func (dao *StakeDao) Name() string {
	return basedef.SERVER_STAKE
}
