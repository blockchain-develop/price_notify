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
	"price_notify/basedef"
	"price_notify/models"
)

type StakeDao struct {
}

func NewStakeDao() *StakeDao {
	stakeDao := &StakeDao{}
	return stakeDao
}

func (dao *StakeDao) AddNotifies([]*models.PriceNotify) error {
	return nil
}

func (dao *StakeDao) GetNotifies() ([]*models.PriceNotify, error) {
	return nil, nil
}

func (dao *StakeDao) Name() string {
	return basedef.SERVER_STAKE
}
