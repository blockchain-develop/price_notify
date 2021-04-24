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
	"io/ioutil"
	"net/http"
	"price_notify/conf"
	"price_notify/models"
	"price_notify/pricenotifydao"
	"runtime/debug"
	"strings"
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

type Notify struct {
	TokenName string
	NotifyPrice int64
	CurrentPrice int64
	Ind int64
}

type PriceNotify struct {
	PriceNotifySlot int64
	Cfg *conf.PriceNotifyConfig
	Notifies        map[string]*Notify
	exit            chan bool
	db              pricenotifydao.PriceNotifyDao
}

func NewPriceNotify(priceNotifySlot int64, priceNotifyCfg *conf.PriceNotifyConfig, db pricenotifydao.PriceNotifyDao) *PriceNotify {
	priceNotify := &PriceNotify{}
	priceNotify.PriceNotifySlot = priceNotifySlot
	priceNotify.Cfg = priceNotifyCfg
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

func (cpl *PriceNotify) findNotifies(priceNotifies []*models.PriceNotify) (map[string]*Notify, error) {
	notifies := make(map[string]*Notify, 0)
	for _, priceNotify := range priceNotifies {
		tokenPrice := priceNotify.TokenBasic.Price
		notify, ok := notifies[priceNotify.TokenBasicName]
		if !ok {
			notify := &Notify{
				TokenName:   priceNotify.TokenBasicName,
				NotifyPrice: priceNotify.Price,
				CurrentPrice: tokenPrice,
			}
			notifies[priceNotify.TokenBasicName] = notify
			if notify.NotifyPrice > tokenPrice {
				notify.NotifyPrice -= 1
				notify.Ind = -1
			} else {
				notify.NotifyPrice += 1
				notify.Ind = 1
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
				notify.CurrentPrice = tokenPrice
				if notify.NotifyPrice > tokenPrice {
					notify.NotifyPrice -= 1
					notify.Ind = -1
				} else {
					notify.NotifyPrice += 1
					notify.Ind = 1
				}
			}
		}
	}
	return notifies, nil
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
		precent, _ := cpl.pricePercent(notify.CurrentPrice, notify.NotifyPrice)
		if precent < 10 {
			cpl.Notifies[notify.TokenName] = notify
		}
	}
	return nil
}

func (cpl *PriceNotify) checkNotifies(priceNotifies []*models.PriceNotify) error {
	notifies, err := cpl.findNotifies(priceNotifies)
	if err != nil {
		return err
	}
	newNotifies := make([]*Notify, 0)
	for _, notify := range notifies {
		percent, _ := cpl.pricePercent(notify.CurrentPrice, notify.NotifyPrice)
		if percent > 10 {
			_, ok := cpl.Notifies[notify.TokenName]
			if ok {
				delete(cpl.Notifies, notify.TokenName)
			}
		}
		if percent < 2 {
			_, ok := cpl.Notifies[notify.TokenName]
			if !ok {
				newNotifies = append(newNotifies, notify)
				cpl.Notifies[notify.TokenName] = notify
			}
			/*
			if ok && notify.Ind != oldNotify.Ind {
				newNotifies = append(newNotifies, notify)
				cpl.Notifies[notify.TokenName] = notify
			}
			*/
		}
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

type DingContent struct {
	Content string `json:"content"`
}
type DingAt struct {
	IsAtAll bool `json:"isAtAll"`
}
type DingNotify struct {
	MsgType string `json:"msgtype"`
	Text DingContent `json:"text"`
	At DingAt `json:"at"`
}

type DingResult struct {
	ErrCode int64 `json:"errcode"`
	ErrMsg string `json:"errmsg"`
}

func (cpl *PriceNotify) notify(notify *Notify) error {
	tag := "above"
	if notify.Ind == -1 {
		tag = "below"
	}
	dingText := fmt.Sprintf("%s price is %s %d", notify.TokenName, tag, notify.NotifyPrice / 100000000)
	dingNotify := &DingNotify{
		MsgType: "text",
		Text:    DingContent{
			Content : dingText,
		},
		At:      DingAt{
			IsAtAll: true,
		},
	}
	url := cpl.Cfg.Node.Url+"robot/send?access_token="+cpl.Cfg.Node.Key
	requestJson, _ := json.Marshal(dingNotify)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(requestJson)))
	if err != nil {
		return err
	}

	req.Header.Set("Accepts", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("response status code: %d", resp.StatusCode)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	dingResult := new(DingResult)
	err = json.Unmarshal(respBody, dingResult)
	if err != nil {
		return err
	}
	if dingResult.ErrCode != 0 || dingResult.ErrMsg != "ok" {
		return fmt.Errorf("code: %d, err: %s", dingResult.ErrCode, dingResult.ErrMsg)
	}
	return nil
}
