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
	Ind int64
}

type PriceNotify struct {
	priceNotifySlot int64
	cfg *conf.PriceNotifyConfig
	exit            chan bool
	notifies map[string]*Trigger
	db              pricenotifydao.PriceNotifyDao
	dingSdk         *dingsdk.DingSdk
}

func NewPriceNotify(priceNotifySlot int64, priceNotifyCfg *conf.PriceNotifyConfig, db pricenotifydao.PriceNotifyDao) *PriceNotify {
	priceNotify := &PriceNotify{}
	priceNotify.priceNotifySlot = priceNotifySlot
	priceNotify.cfg = priceNotifyCfg
	priceNotify.notifies = make(map[string]*Trigger, 0)
	priceNotify.db = db
	priceNotify.exit = make(chan bool, 0)
	priceNotify.dingSdk = dingsdk.NewDingSdk(priceNotifyCfg.Node.Url, priceNotifyCfg.Node.Key)
	//
	tokens, err := db.GetTokens()
	if err != nil {
		panic(err)
	}
	err = priceNotify.checkNotifies(tokens)
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
			if now%cpl.priceNotifySlot != 0 {
				continue
			}

			logs.Info("do price notify at time: %s", time.Now().Format("2006-01-02 15:04:05"))
			tokens, err := cpl.db.GetTokens()
			if err != nil {
				logs.Error("get price notify err: %v", err)
				continue
			}
			err = cpl.checkNotifies(tokens)
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

func (cpl *PriceNotify) checkNotifies(tokens []*models.TokenBasic) error {
	newNotifies := make([]*Trigger, 0)
	for _, token := range tokens {
		notify, ok := cpl.notifies[token.Name]
		if !ok {
			notify = &Trigger{
				TokenName:   token.Name,
				NotifyPrice: token.Price,
				Ind:         1,
			}
			newNotifies = append(newNotifies, notify)
			cpl.notifies[token.Name] = notify
		} else {
			percent, ind := cpl.pricePercent(token.Price, notify.NotifyPrice)
			if percent > 10 {
				notify.NotifyPrice = token.Price
				notify.Ind = ind
				newNotifies = append(newNotifies, notify)
			}
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
	if notify.Ind == -1 {
		tag = "down"
	}
	price := decimal.NewFromInt(notify.NotifyPrice)
	newPrice := price.Div(decimal.NewFromInt(100000000))
	dingText := fmt.Sprintf("%s price is %s to %s", notify.TokenName, tag, newPrice.String())
	dingNotify := &dingsdk.DingNotify{
		MsgType: "text",
		Text:    dingsdk.DingContent{
			Content : dingText,
		},
		At:      dingsdk.DingAt{
			IsAtAll: false,
		},
	}
	if cpl.cfg.Switch == false {
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
