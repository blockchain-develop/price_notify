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
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/shopspring/decimal"
	"price_notify/conf"
	"price_notify/dingsdk"
	"price_notify/models"
	"price_notify/pricenotifydao"
	"runtime/debug"
	"time"
)

var priceNotify *PriceNotify

func StartPriceNotify(server string, priceNotifySlot int64, priceNotifyCfg *conf.PriceNotifyConfig, dbCfg *conf.DBConfig) {
	dao := pricenotifydao.NewPriceNotifyDao(server, dbCfg)
	if dao == nil {
		panic("server is not valid")
	}
	priceNotify = NewPriceNotify(priceNotifySlot, priceNotifyCfg, dao)
	priceNotify.Start()
}

func StopPriceNotify() {
	if priceNotify != nil {
		priceNotify.Stop()
	}
}

type Trigger struct {
	TokenName string
	NotifyPrice int64
	CurrentPrice int64
}

type PriceNotify struct {
	PriceNotifySlot int64
	Cfg *conf.PriceNotifyConfig
	triggers        map[string]*Trigger
	exit            chan bool
	db              pricenotifydao.PriceNotifyDao
	dingSdk         *dingsdk.DingSdk
}

func NewPriceNotify(priceNotifySlot int64, priceNotifyCfg *conf.PriceNotifyConfig, db pricenotifydao.PriceNotifyDao) *PriceNotify {
	priceNotify := &PriceNotify{}
	priceNotify.PriceNotifySlot = priceNotifySlot
	priceNotify.Cfg = priceNotifyCfg
	priceNotify.triggers = make(map[string]*Trigger, 0)
	priceNotify.db = db
	priceNotify.exit = make(chan bool, 0)
	priceNotify.dingSdk = dingsdk.NewDingSdk(priceNotifyCfg.Node.Url, priceNotifyCfg.Node.Key)
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

func (cpl *PriceNotify) findNotifies(priceNotifies []*models.PriceNotify) (map[string]*Trigger, error) {
	triggers := make(map[string]*Trigger, 0)
	for _, priceNotify := range priceNotifies {
		tokenPrice := priceNotify.TokenBasic.Price
		trigger, ok := triggers[priceNotify.TokenBasicName]
		if !ok {
			trigger = &Trigger{
				TokenName:   priceNotify.TokenBasicName,
				NotifyPrice: priceNotify.Price,
				CurrentPrice: tokenPrice,
			}
			triggers[priceNotify.TokenBasicName] = trigger
		} else {
			newDiff := priceNotify.Price - tokenPrice
			oldDiff := trigger.NotifyPrice - tokenPrice
			if newDiff < 0 {
				newDiff = 0 - newDiff
			}
			if oldDiff < 0 {
				oldDiff = 0 - oldDiff
			}
			if newDiff < oldDiff {
				trigger.NotifyPrice = priceNotify.Price
				trigger.CurrentPrice = tokenPrice
			}
		}
	}
	return triggers, nil
}

func (cpl *PriceNotify) initNotifies(priceNotifies []*models.PriceNotify) error {
	notifies, err := cpl.findNotifies(priceNotifies)
	if err != nil {
		return err
	}
	for _, notify := range notifies {
		cpl.notify(notify)
	}
	for _, notify := range notifies {
		percent, _ := cpl.pricePercent(notify.CurrentPrice, notify.NotifyPrice)
		if percent < 10 {
			cpl.triggers[notify.TokenName] = notify
		}
	}
	return nil
}

func (cpl *PriceNotify) checkNotifies(priceNotifies []*models.PriceNotify) error {
	notifies, err := cpl.findNotifies(priceNotifies)
	if err != nil {
		return err
	}
	newNotifies := make([]*Trigger, 0)
	for _, notify := range notifies {
		{
			oldNotify, ok := cpl.triggers[notify.TokenName]
			if ok {
				if oldNotify.NotifyPrice != notify.NotifyPrice {
					newNotifies = append(newNotifies, notify)
					cpl.triggers[notify.TokenName] = notify
				}
			}
		}
		percent, _ := cpl.pricePercent(notify.CurrentPrice, notify.NotifyPrice)
		if percent > 10 {
			_, ok := cpl.triggers[notify.TokenName]
			if ok {
				newNotifies = append(newNotifies, notify)
				delete(cpl.triggers, notify.TokenName)
				/*
				if (oldNotify.CurrentPrice - oldNotify.NotifyPrice) * (notify.CurrentPrice - notify.NotifyPrice) < 0 {
					delete(cpl.triggers, notify.TokenName)
				} else {
					newNotifies = append(newNotifies, notify)
					delete(cpl.triggers, notify.TokenName)
				}
				*/
			}
		}
		if percent < 1 {
			_, ok := cpl.triggers[notify.TokenName]
			if !ok {
				newNotifies = append(newNotifies, notify)
				cpl.triggers[notify.TokenName] = notify
			}
			/*
			if ok && notify.Ind != oldNotify.Ind {
				newNotifies = append(newNotifies, notify)
				cpl.Notifies[notify.TokenName] = notify
			}
			*/
		}
	}
	for _, notify := range newNotifies {
		cpl.notify(notify)
	}
	return nil
}

func (cpl *PriceNotify) pricePercent(price int64, base int64) (int64, int64) {
	ind := int64(0)
	diff := price - base
	if diff < 0 {
		ind = -1
		diff = 0 - diff
	} else {
		ind = 1
	}
	percent := diff * 1000 / base
	return percent, ind
}

func (cpl *PriceNotify) notify(notify *Trigger) error {
	tag := "up"
	percent, _ := cpl.pricePercent(notify.CurrentPrice, notify.NotifyPrice)
	if percent < 1 {
		if notify.CurrentPrice - notify.NotifyPrice > 0 {
			tag = "down"
		}
	} else {
		if notify.CurrentPrice - notify.NotifyPrice < 0 {
			tag = "down"
		}
	}
	price := decimal.NewFromInt(notify.CurrentPrice)
	newPrice := price.Div(decimal.NewFromInt(100000000))
	dingText := fmt.Sprintf("%s price is %s to %s", notify.TokenName, tag, newPrice.String())
	dingNotify := &dingsdk.DingNotify{
		MsgType: "text",
		Text:    dingsdk.DingContent{
			Content : dingText,
		},
		At:      dingsdk.DingAt{
			IsAtAll: true,
		},
	}
	if cpl.Cfg.Switch == false {
		notifyJson, _ := json.Marshal(dingNotify)
		logs.Info("ding notify: %s", string(notifyJson))
		return nil
	}
	result, err := cpl.dingSdk.Notify(dingNotify)
	if err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("code: %d, err: %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}
