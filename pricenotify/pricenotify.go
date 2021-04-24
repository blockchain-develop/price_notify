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

package pricenotify

import (
	"github.com/astaxie/beego/logs"
	"price_notify/conf"
	"price_notify/models"
	"price_notify/pricenotifydao"
	"runtime/debug"
	"time"
)

var priceNotify *PriceNotify

func StartPriceNotify(server string, priceNotifySlot int64, dbCfg *conf.DBConfig) {
	dao := pricenotifydao.NewPriceNotifyDao(server, dbCfg)
	if dao == nil {
		panic("server is not valid")
	}
	priceNotify = NewPriceNotify(priceNotifySlot, dao)
	priceNotify.Start()
}

func StopPriceNotify() {
	if priceNotify != nil {
		priceNotify.Stop()
	}
}

type Notify struct {
	TokenName string
	NotifyPrice int64
}

type PriceNotify struct {
	PriceNotifySlot int64
	Notifies        map[string]*Notify
	exit            chan bool
	db              pricenotifydao.PriceNotifyDao
}

func NewPriceNotify(priceNotifySlot int64, db pricenotifydao.PriceNotifyDao) *PriceNotify {
	priceNotify := &PriceNotify{}
	priceNotify.PriceNotifySlot = priceNotifySlot
	priceNotify.Notifies = make(map[string]*Notify, 0)
	priceNotify.db = db
	priceNotify.exit = make(chan bool, 0)
	//
	notifies, err := db.GetNotifies()
	if err != nil {
		panic(err)
	}
	err = priceNotify.initNotifies(notifies)
	if err != nil {
		panic(err)
	}
	return priceNotify
}

func (cpl *PriceNotify) Start() {
	logs.Info("start price notify.")
	go cpl.PriceNotify()
}

func (cpl *PriceNotify) Stop() {
	cpl.exit <- true
	logs.Info("stop price notify.")
}

func (cpl *PriceNotify) PriceNotify() {
	for {
		exit := cpl.priceNotify()
		if exit {
			close(cpl.exit)
			break
		}
		time.Sleep(time.Second * 5)
	}
}

func (cpl *PriceNotify) priceNotify() (exit bool) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("service start, recover info: %s", string(debug.Stack()))
			exit = false
		}
	}()

	logs.Debug("price notify, dao: %s......", cpl.db.Name())
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			if now%cpl.PriceNotifySlot != 0 {
				continue
			}

			logs.Info("do price notify at time: %s", time.Now().Format("2006-01-02 15:04:05"))
			notifies, err := cpl.db.GetNotifies()
			if err != nil {
				logs.Error("get price notify err: %v", err)
				continue
			}
			err = cpl.checkNotifies(notifies)
			if err != nil {
				logs.Error("check price notify err: %v", err)
				continue
			}
			break
		case <-cpl.exit:
			logs.Info("coin price listen exit, dao: %s......", cpl.db.Name())
			return true
		}
	}
}

func (cpl *PriceNotify) initNotifies(priceNotifies []*models.PriceNotify) error {
	for _, priceNotify := range priceNotifies {
		tokenPrice := priceNotify.TokenBasic.Price
		notify, ok := cpl.Notifies[priceNotify.TokenBasicName]
		if !ok {
			notify := &Notify{
				TokenName:   priceNotify.TokenBasicName,
				NotifyPrice: priceNotify.Price,
			}
			cpl.Notifies[priceNotify.TokenBasicName] = notify
			if notify.NotifyPrice > tokenPrice {
				notify.NotifyPrice -= 1
			} else {
				notify.NotifyPrice += 1
			}
		} else {
			newDiff := priceNotify.Price - tokenPrice
			oldDiff := notify.NotifyPrice - tokenPrice
			if newDiff < 0 {
				newDiff = 0 - newDiff
			}
			if oldDiff < 0 {
				oldDiff = 0 - oldDiff
			}
			if newDiff < oldDiff {
				notify.NotifyPrice = priceNotify.Price
				if notify.NotifyPrice > tokenPrice {
					notify.NotifyPrice -= 1
				} else {
					notify.NotifyPrice += 1
				}
			}
		}
	}
	return nil
}

func (cpl *PriceNotify) checkNotifies(priceNotifies []*models.PriceNotify) error {
	for _, priceNotify := range priceNotifies {
		tokenPrice := priceNotify.TokenBasic.Price
		notify, ok := cpl.Notifies[priceNotify.TokenBasicName]
		if !ok {
			notify := &Notify{
				TokenName:   priceNotify.TokenBasicName,
				NotifyPrice: priceNotify.Price,
			}
			cpl.Notifies[priceNotify.TokenBasicName] = notify
			if notify.NotifyPrice > tokenPrice {
				notify.NotifyPrice -= 1
			} else {
				notify.NotifyPrice += 1
			}
		} else {
			newDiff := priceNotify.Price - tokenPrice
			oldDiff := notify.NotifyPrice - tokenPrice
			if newDiff < 0 {
				newDiff = 0 - newDiff
			}
			if oldDiff < 0 {
				oldDiff = 0 - oldDiff
			}
			if newDiff < oldDiff {
				notify.NotifyPrice = priceNotify.Price
				if notify.NotifyPrice > tokenPrice {
					notify.NotifyPrice -= 1
				} else {
					notify.NotifyPrice += 1
				}
			}
		}
	}
	return nil
}
