/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package agents

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/radigo"
	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/avp"
	"github.com/fiorix/go-diameter/diam/datatype"
)

func TestAgReqSetFields(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.CGRID},
		utils.Sha1("dsafdsaf", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()),
		false, false)
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)
	agReq.CGRRequest.Set([]string{utils.AnswerTime},
		time.Date(2013, 12, 30, 15, 0, 1, 0, time.UTC), false, false)
	agReq.CGRRequest.Set([]string{utils.RequestType}, utils.META_PREPAID, false, false)
	agReq.CGRRequest.Set([]string{utils.Usage}, time.Duration(3*time.Minute), false, false)

	cgrRply := map[string]interface{}{
		utils.CapAttributes: map[string]interface{}{
			"PaypalAccount": "cgrates@paypal.com",
		},
		utils.CapMaxUsage: time.Duration(120 * time.Second),
		utils.Error:       "",
	}
	agReq.CGRReply = config.NewNavigableMap(cgrRply)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaRep + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaRep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Destination",
			Path: utils.MetaRep + utils.NestingSep + utils.Destination, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},

		&config.FCTemplate{Tag: "RequestedUsageVoice",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*voice"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_seconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "RequestedUsageData",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*data"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_nanoseconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "RequestedUsageSMS",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*sms"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_nanoseconds}", true, utils.INFIELD_SEP)},

		&config.FCTemplate{Tag: "AttrPaypalAccount",
			Path: utils.MetaRep + utils.NestingSep + "PaypalAccount", Type: utils.MetaVariable,
			Filters: []string{"*empty:~*cgrep.Error:"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.Attributes.PaypalAccount", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "MaxUsage",
			Path: utils.MetaRep + utils.NestingSep + "MaxUsage", Type: utils.MetaVariable,
			Filters: []string{"*empty:~*cgrep.Error:"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.MaxUsage{*duration_seconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Error",
			Path: utils.MetaRep + utils.NestingSep + "Error", Type: utils.MetaVariable,
			Filters: []string{"*rsr::~*cgrep.Error(!^$)"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.Error", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	eMp.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[2]}}, false, true)
	eMp.Set([]string{"RequestedUsage"}, []*config.NMItem{
		&config.NMItem{Data: "180", Path: []string{"RequestedUsage"},
			Config: tplFlds[3]}}, false, true)
	eMp.Set([]string{"PaypalAccount"}, []*config.NMItem{
		&config.NMItem{Data: "cgrates@paypal.com", Path: []string{"PaypalAccount"},
			Config: tplFlds[6]}}, false, true)
	eMp.Set([]string{"MaxUsage"}, []*config.NMItem{
		&config.NMItem{Data: "120", Path: []string{"MaxUsage"},
			Config: tplFlds[7]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.Reply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.Reply)
	}
}

func TestAgentRequestSetFields(t *testing.T) {
	req := map[string]interface{}{
		utils.Account: 1009,
		utils.Tenant:  "cgrates.org",
	}
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	vars := map[string]interface{}{}
	ar := NewAgentRequest(config.NewNavigableMap(req), vars,
		nil, nil, config.NewRSRParsersMustCompile("", false, utils.NestingSep),
		"cgrates.org", "", engine.NewFilterS(cfg, nil, dm),
		config.NewNavigableMap(req), config.NewNavigableMap(req))
	input := []*config.FCTemplate{}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	}
	// tplFld.Type == utils.META_NONE
	input = []*config.FCTemplate{&config.FCTemplate{Type: utils.META_NONE}}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	}
	// unsupported type: <>
	input = []*config.FCTemplate{&config.FCTemplate{Blocker: true}}
	if err := ar.SetFields(input); err == nil || err.Error() != "unsupported type: <>" {
		t.Error(err)
	}
	// case utils.MetaVars
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaVars),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// case utils.MetaCgreq
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaCgreq),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaCgreq),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.CGRRequest.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// case utils.MetaCgrep
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaCgrep),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaCgrep),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.CGRReply.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// case utils.MetaRep
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaRep),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaRep),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Reply.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// case utils.MetaDiamreq
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaDiamreq),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaDiamreq),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.diamreq.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	//META_COMPOSED
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Type:  utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Tenant", false, ";"),
		},
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Type:  utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile(":", false, ";"),
		},
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Type:  utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"AccountID"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "cgrates.org:1009" {
		t.Error("Expecting 'cgrates.org:1009', received: ", nm[0].Data)
	}

	// META_CONSTANT
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Account", utils.MetaVars),
			Type:  utils.META_CONSTANT,
			Value: config.NewRSRParsersMustCompile("2020", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"Account"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "2020" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// Filters
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:    fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Tag:     fmt.Sprintf("%s.AccountID", utils.MetaVars),
			Filters: []string{utils.MetaString + ":~" + utils.MetaVars + ".Account:1003"},
			Type:    utils.META_CONSTANT,
			Value:   config.NewRSRParsersMustCompile("2021", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"AccountID"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item ", utils.ToJSON(nm))
	} else if nm[0].Data != "cgrates.org:1009" {
		t.Error("Expecting 'cgrates.org:1009', received: ", nm[0].Data)
	}

	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:    fmt.Sprintf("%s.Account", utils.MetaVars),
			Tag:     fmt.Sprintf("%s.Account", utils.MetaVars),
			Filters: []string{"Not really a filter"},
			Type:    utils.META_CONSTANT,
			Value:   config.NewRSRParsersMustCompile("2021", false, ";"),
		},
	}
	if err := ar.SetFields(input); err == nil || err.Error() != "NOT_FOUND:Not really a filter" {
		t.Errorf("Expecting: 'NOT_FOUND:Not really a filter', received: %+v", err)
	}

	// Blocker: true
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:    fmt.Sprintf("%s.Name", utils.MetaVars),
			Tag:     fmt.Sprintf("%s.Name", utils.MetaVars),
			Type:    utils.MetaVariable,
			Value:   config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Account", false, ";"),
			Blocker: true,
		},
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Name", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Name", utils.MetaVars),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("1005", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"Name"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	// ErrNotFound
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Test", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Test", utils.MetaVars),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Test", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if _, err := ar.Vars.GetField([]string{"Test"}); err == nil || err != utils.ErrNotFound {
		t.Errorf("Expecting: %+v, received: %+v", utils.ErrNotFound, err)
	}
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:      fmt.Sprintf("%s.Test", utils.MetaVars),
			Tag:       fmt.Sprintf("%s.Test", utils.MetaVars),
			Type:      utils.MetaVariable,
			Value:     config.NewRSRParsersMustCompile("~"+utils.MetaReq+".Test", false, ";"),
			Mandatory: true,
		},
	}
	if err := ar.SetFields(input); err == nil || err.Error() != "NOT_FOUND:"+utils.MetaVars+".Test" {
		t.Errorf("Expecting: %+v, received: %+v", "NOT_FOUND:"+utils.MetaVars+".Test", err)
	}

	//Not found
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:      "wrong",
			Tag:       "wrong",
			Type:      utils.MetaVariable,
			Value:     config.NewRSRParsersMustCompile("~*req.Account", false, ";"),
			Mandatory: true,
		},
	}
	if err := ar.SetFields(input); err == nil || err.Error() != "unsupported field prefix: <wrong>" {
		t.Errorf("Expecting: %+v, received: %+v", "unsupported field prefix: <wrong>", err)
	}

	// MetaHdr/MetaTrl
	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account4", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Account4", utils.MetaVars),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaHdr+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"Account4"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}

	input = []*config.FCTemplate{
		&config.FCTemplate{
			Path:  fmt.Sprintf("%s.Account5", utils.MetaVars),
			Tag:   fmt.Sprintf("%s.Account5", utils.MetaVars),
			Type:  utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~"+utils.MetaTrl+".Account", false, ";"),
		},
	}
	if err := ar.SetFields(input); err != nil {
		t.Error(err)
	} else if val, err := ar.Vars.GetField([]string{"Account5"}); err != nil {
		t.Error(err)
	} else if nm, ok := val.([]*config.NMItem); !ok {
		t.Error("Expecting NM items")
	} else if len(nm) != 1 {
		t.Error("Expecting one item")
	} else if nm[0].Data != "1009" {
		t.Error("Expecting 1009, received: ", nm[0].Data)
	}
}

func TestAgReqMaxCost(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.CapMaxUsage}, "120s", false, false)

	cgrRply := map[string]interface{}{
		utils.CapMaxUsage: time.Duration(120 * time.Second),
	}
	agReq.CGRReply = config.NewNavigableMap(cgrRply)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "MaxUsage",
			Path: utils.MetaRep + utils.NestingSep + "MaxUsage", Type: utils.MetaVariable,
			Filters: []string{"*rsr::~*cgrep.MaxUsage(>0s)"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.MaxUsage{*duration_seconds}", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)

	eMp.Set([]string{"MaxUsage"}, []*config.NMItem{
		&config.NMItem{Data: "120", Path: []string{"MaxUsage"},
			Config: tplFlds[0]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.Reply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.Reply)
	}
}

func TestAgReqParseFieldDiameter(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "MandatoryFalse",
			Path: "MandatoryFalse", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryFalse", true, utils.INFIELD_SEP),
			Mandatory: false},
		&config.FCTemplate{Tag: "MandatoryTrue",
			Path: "MandatoryTrue", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryTrue", true, utils.INFIELD_SEP),
			Mandatory: true},
		&config.FCTemplate{Tag: "Session-Id", Filters: []string{},
			Path: "Session-Id", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := ""
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
	if _, err := agReq.ParseField(tplFlds[1]); err == nil ||
		err.Error() != "Empty source value for fieldID: <MandatoryTrue>" {
		t.Error(err)
	}
	expected = "simuhuawei;1449573472;00002"
	if out, err := agReq.ParseField(tplFlds[2]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
}

func TestAgReqParseFieldRadius(t *testing.T) {
	//creater radius message
	pkt := radigo.NewPacket(radigo.AccountingRequest, 1, dictRad, coder, "CGRateS.org")
	if err := pkt.AddAVPWithName("User-Name", "flopsy", ""); err != nil {
		t.Error(err)
	}
	if err := pkt.AddAVPWithName("Cisco-NAS-Port", "CGR1", "Cisco"); err != nil {
		t.Error(err)
	}
	//create radiusDataProvider
	dP := newRADataProvider(pkt)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "MandatoryFalse",
			Path: "MandatoryFalse", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryFalse", true, utils.INFIELD_SEP),
			Mandatory: false},
		&config.FCTemplate{Tag: "MandatoryTrue",
			Path: "MandatoryTrue", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryTrue", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := ""
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
	if _, err := agReq.ParseField(tplFlds[1]); err == nil ||
		err.Error() != "Empty source value for fieldID: <MandatoryTrue>" {
		t.Error(err)
	}
}

func TestAgReqParseFieldHttpUrl(t *testing.T) {
	//creater radius message
	br := bufio.NewReader(strings.NewReader(`GET /cdr?request_type=MOSMS_CDR&timestamp=2008-08-15%2017:49:21&message_date=2008-08-15%2017:49:21&transactionid=100744&CDR_ID=123456&carrierid=1&mcc=222&mnc=10&imsi=235180000000000&msisdn=%2B4977000000000&destination=%2B497700000001&message_status=0&IOT=0&service_id=1 HTTP/1.1
Host: api.cgrates.org

`))
	req, err := http.ReadRequest(br)
	if err != nil {
		t.Error(err)
	}
	//create radiusDataProvider
	dP, _ := newHTTPUrlDP(req)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "MandatoryFalse",
			Path: "MandatoryFalse", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryFalse", true, utils.INFIELD_SEP),
			Mandatory: false},
		&config.FCTemplate{Tag: "MandatoryTrue",
			Path: "MandatoryTrue", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryTrue", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := ""
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}

	if _, err := agReq.ParseField(tplFlds[1]); err == nil ||
		err.Error() != "Empty source value for fieldID: <MandatoryTrue>" {
		t.Error(err)
	}
}

func TestAgReqParseFieldHttpXml(t *testing.T) {
	//creater radius message
	body := `<complete-success-notification callid="109870">
	<createtime>2005-08-26T14:16:42</createtime>
	<connecttime>2005-08-26T14:16:56</connecttime>
	<endtime>2005-08-26T14:17:34</endtime>
	<reference>My Call Reference</reference>
	<userid>386</userid>
	<username>sampleusername</username>
	<customerid>1</customerid>
	<companyname>Conecto LLC</companyname>
	<totalcost amount="0.21" currency="USD">US$0.21</totalcost>
	<hasrecording>yes</hasrecording>
	<hasvoicemail>no</hasvoicemail>
	<agenttotalcost amount="0.13" currency="USD">US$0.13</agenttotalcost>
	<agentid>44</agentid>
	<callleg calllegid="222146">
		<number>+441624828505</number>
		<description>Isle of Man</description>
		<seconds>38</seconds>
		<perminuterate amount="0.0200" currency="USD">US$0.0200</perminuterate>
		<cost amount="0.0140" currency="USD">US$0.0140</cost>
		<agentperminuterate amount="0.0130" currency="USD">US$0.0130</agentperminuterate>
		<agentcost amount="0.0082" currency="USD">US$0.0082</agentcost>
	</callleg>
	<callleg calllegid="222147">
		<number>+44 7624 494075</number>
		<description>Isle of Man</description>
		<seconds>37</seconds>
		<perminuterate amount="0.2700" currency="USD">US$0.2700</perminuterate>
		<cost amount="0.1890" currency="USD">US$0.1890</cost>
		<agentperminuterate amount="0.1880" currency="USD">US$0.1880</agentperminuterate>
		<agentcost amount="0.1159" currency="USD">US$0.1159</agentcost>
	</callleg>
</complete-success-notification>
`
	req, err := http.NewRequest("POST", "http://localhost:8080/", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Error(err)
	}
	//create radiusDataProvider
	dP, _ := newHTTPXmlDP(req)
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)

	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "MandatoryFalse",
			Path: "MandatoryFalse", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryFalse", true, utils.INFIELD_SEP),
			Mandatory: false},
		&config.FCTemplate{Tag: "MandatoryTrue",
			Path: "MandatoryTrue", Type: utils.META_COMPOSED,
			Value:     config.NewRSRParsersMustCompile("~*req.MandatoryTrue", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := ""
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
	if _, err := agReq.ParseField(tplFlds[1]); err == nil ||
		err.Error() != "Empty source value for fieldID: <MandatoryTrue>" {
		t.Error(err)
	}
}

func TestAgReqEmptyFilter(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.CGRID},
		utils.Sha1("dsafdsaf", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()),
		false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant", Filters: []string{},
			Path: utils.MetaCgrep + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},

		&config.FCTemplate{Tag: "Account", Filters: []string{},
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Destination", Filters: []string{},
			Path: utils.MetaCgrep + utils.NestingSep + utils.Destination, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	eMp.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[2]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRReply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRReply)
	}
}

func TestAgReqMetaExponent(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	agReq.CGRRequest.Set([]string{"Value"}, "2", false, false)
	agReq.CGRRequest.Set([]string{"Exponent"}, "2", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "TestExpo", Filters: []string{},
			Path: utils.MetaCgrep + utils.NestingSep + "TestExpo", Type: utils.MetaValueExponent,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Value;~*cgreq.Exponent", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{"TestExpo"}, []*config.NMItem{
		&config.NMItem{Data: "200", Path: []string{"TestExpo"},
			Config: tplFlds[0]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRReply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRReply)
	}
}

func TestAgReqFieldAsNone(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Type: utils.META_NONE, Blocker: true},
		&config.FCTemplate{Tag: "Destination",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Destination, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRReply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRReply)
	}
}

func TestAgReqFieldAsNone2(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Type: utils.META_NONE},
		&config.FCTemplate{Tag: "Destination",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Destination, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	eMp.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[3]}}, false, true)
	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRReply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRReply)
	}
}

func TestAgReqSetField2(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)
	agReq.CGRRequest.Set([]string{utils.AnswerTime},
		time.Date(2013, 12, 30, 15, 0, 1, 0, time.UTC), false, false)
	agReq.CGRRequest.Set([]string{utils.RequestType}, utils.META_PREPAID, false, false)

	agReq.CGRReply = config.NewNavigableMap(nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Tenant, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Destination",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Destination, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Usage",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Usage, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("30s", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "CalculatedUsage",
			Path: utils.MetaCgrep + utils.NestingSep + "CalculatedUsage",
			Type: "*difference", Value: config.NewRSRParsersMustCompile("~*cgreq.AnswerTime;~*cgrep.Usage", true, utils.INFIELD_SEP),
		},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	eMp.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[2]}}, false, true)
	eMp.Set([]string{"Usage"}, []*config.NMItem{
		&config.NMItem{Data: "30s", Path: []string{"Usage"},
			Config: tplFlds[3]}}, false, true)
	eMp.Set([]string{"CalculatedUsage"}, []*config.NMItem{
		&config.NMItem{Data: time.Date(2013, 12, 30, 14, 59, 31, 0, time.UTC), Path: []string{"CalculatedUsage"},
			Config: tplFlds[4]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRReply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRReply)
	}
}

func TestAgReqFieldAsInterface(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest = config.NewNavigableMap(nil)
	agReq.CGRRequest.Set([]string{utils.Usage}, []*config.NMItem{{Data: 3 * time.Minute}}, false, false)
	agReq.CGRRequest.Set([]string{utils.ToR}, []*config.NMItem{{Data: utils.VOICE}}, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)

	path := []string{utils.MetaCgreq, utils.Usage}
	var expVal interface{}
	expVal = []*config.NMItem{{Data: 3 * time.Minute}}
	if rply, err := agReq.FieldAsInterface(path); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rply, expVal) {
		t.Errorf("Expected %v , received: %v", utils.ToJSON(expVal), utils.ToJSON(rply))
	}

	path = []string{utils.MetaCgreq, utils.ToR}
	expVal = []*config.NMItem{{Data: utils.VOICE}}
	if rply, err := agReq.FieldAsInterface(path); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rply, expVal) {
		t.Errorf("Expected %v , received: %v", utils.ToJSON(expVal), utils.ToJSON(rply))
	}

	path = []string{utils.MetaCgreq, utils.Account}
	expVal = "1001"
	if rply, err := agReq.FieldAsInterface(path); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rply, expVal) {
		t.Errorf("Expected %v , received: %v", utils.ToJSON(expVal), utils.ToJSON(rply))
	}

	path = []string{utils.MetaCgreq, utils.Destination}
	expVal = "1002"
	if rply, err := agReq.FieldAsInterface(path); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rply, expVal) {
		t.Errorf("Expected %v , received: %v", utils.ToJSON(expVal), utils.ToJSON(rply))
	}
}

func TestAgReqNewARWithCGRRplyAndRply(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)

	ev := map[string]interface{}{
		"FirstLevel": map[string]interface{}{
			"SecondLevel": map[string]interface{}{
				"Fld1": "Val1",
			},
		},
	}
	rply := config.NewNavigableMap(ev)

	ev2 := map[string]interface{}{
		utils.CapAttributes: map[string]interface{}{
			"PaypalAccount": "cgrates@paypal.com",
		},
		utils.CapMaxUsage: time.Duration(120 * time.Second),
		utils.Error:       "",
	}
	cgrRply := config.NewNavigableMap(ev2)

	agReq := NewAgentRequest(nil, nil, cgrRply, rply, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Fld1",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld1", Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*rep.FirstLevel.SecondLevel.Fld1", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Fld2",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld2", Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgrep.Attributes.PaypalAccount", true, utils.INFIELD_SEP)},
	}

	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{"Fld1"}, []*config.NMItem{
		&config.NMItem{Data: "Val1", Path: []string{"Fld1"},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{"Fld2"}, []*config.NMItem{
		&config.NMItem{Data: "cgrates@paypal.com", Path: []string{"Fld2"},
			Config: tplFlds[1]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRRequest, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRRequest)
	}
}

func TestAgReqSetCGRReplyWithError(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)

	ev := map[string]interface{}{
		"FirstLevel": map[string]interface{}{
			"SecondLevel": map[string]interface{}{
				"Fld1": "Val1",
			},
		},
	}
	rply := config.NewNavigableMap(ev)

	agReq := NewAgentRequest(nil, nil, nil, rply, nil, "cgrates.org", "", filterS, nil, nil)

	agReq.setCGRReply(nil, utils.ErrNotFound)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Fld1",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld1", Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*rep.FirstLevel.SecondLevel.Fld1", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Fld2",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld2", Type: utils.MetaVariable,
			Value:     config.NewRSRParsersMustCompile("~*cgrep.Attributes.PaypalAccount", true, utils.INFIELD_SEP),
			Mandatory: true},
	}

	if err := agReq.SetFields(tplFlds); err == nil ||
		err.Error() != "NOT_FOUND:Fld2" {
		t.Error(err)
	}
}

type myEv map[string]interface{}

func (ev myEv) AsNavigableMap(tpl []*config.FCTemplate) (*config.NavigableMap, error) {
	return config.NewNavigableMap(ev), nil
}

func TestAgReqSetCGRReplyWithoutError(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)

	ev := map[string]interface{}{
		"FirstLevel": map[string]interface{}{
			"SecondLevel": map[string]interface{}{
				"Fld1": "Val1",
			},
		},
	}
	rply := config.NewNavigableMap(ev)

	myEv := myEv{
		utils.CapAttributes: map[string]interface{}{
			"PaypalAccount": "cgrates@paypal.com",
		},
		utils.CapMaxUsage: time.Duration(120 * time.Second),
		utils.Error:       "",
	}

	agReq := NewAgentRequest(nil, nil, nil, rply,
		nil, "cgrates.org", "", filterS, nil, nil)

	agReq.setCGRReply(myEv, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Fld1",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld1", Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*rep.FirstLevel.SecondLevel.Fld1", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Fld2",
			Path: utils.MetaCgreq + utils.NestingSep + "Fld2", Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgrep.Attributes.PaypalAccount", true, utils.INFIELD_SEP)},
	}

	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{"Fld1"}, []*config.NMItem{
		&config.NMItem{Data: "Val1", Path: []string{"Fld1"},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{"Fld2"}, []*config.NMItem{
		&config.NMItem{Data: "cgrates@paypal.com", Path: []string{"Fld2"},
			Config: tplFlds[1]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.CGRRequest, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.CGRRequest)
	}
}

func TestAgReqParseFieldMetaCCUsage(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "CCUsage", Filters: []string{},
			Path: "CCUsage", Type: utils.MetaCCUsage,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid arguments <[{"Rules":"~*req.Session-Id","AllFiltersMatch":true}]> to *cc_usage` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "CCUsage", Filters: []string{},
			Path: "CCUsage", Type: utils.MetaCCUsage,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id;12s;12s", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid requestNumber <simuhuawei;1449573472;00002> to *cc_usage` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "CCUsage", Filters: []string{},
			Path: "CCUsage", Type: utils.MetaCCUsage,
			Value:     config.NewRSRParsersMustCompile("10;~*req.Session-Id;12s", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid usedCCTime <simuhuawei;1449573472;00002> to *cc_usage` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "CCUsage", Filters: []string{},
			Path: "CCUsage", Type: utils.MetaCCUsage,
			Value:     config.NewRSRParsersMustCompile("10;12s;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid debitInterval <simuhuawei;1449573472;00002> to *cc_usage` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "CCUsage", Filters: []string{},
			Path: "CCUsage", Type: utils.MetaCCUsage,
			Value:     config.NewRSRParsersMustCompile("3;10s;5s", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	//5s*2 + 10s
	expected := time.Duration(20 * time.Second)
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
}

func TestAgReqParseFieldMetaUsageDifference(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Usage", Filters: []string{},
			Path: "Usage", Type: utils.META_USAGE_DIFFERENCE,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid arguments <[{"Rules":"~*req.Session-Id","AllFiltersMatch":true}]> to *usage_difference` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Usage", Filters: []string{},
			Path: "Usage", Type: utils.META_USAGE_DIFFERENCE,
			Value:     config.NewRSRParsersMustCompile("1560325161;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `Unsupported time format` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Usage", Filters: []string{},
			Path: "Usage", Type: utils.META_USAGE_DIFFERENCE,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id;1560325161", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `Unsupported time format` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Usage", Filters: []string{},
			Path: "Usage", Type: utils.META_USAGE_DIFFERENCE,
			Value:     config.NewRSRParsersMustCompile("1560325161;1560325151", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := "10s"
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, received: <%+v>", expected, out)
	}
}

func TestAgReqParseFieldMetaSum(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Sum", Filters: []string{},
			Path: "Sum", Type: utils.MetaSum,
			Value:     config.NewRSRParsersMustCompile("15;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `strconv.ParseInt: parsing "simuhuawei;1449573472;00002": invalid syntax` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Sum", Filters: []string{},
			Path: "Sum", Type: utils.MetaSum,
			Value:     config.NewRSRParsersMustCompile("15;15", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := int64(30)
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, %T received: <%+v> %T", expected, expected, out, out)
	}
}

func TestAgReqParseFieldMetaDifference(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	dm := engine.NewDataManager(engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items),
		config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Diff", Filters: []string{},
			Path: "Diff", Type: utils.MetaDifference,
			Value:     config.NewRSRParsersMustCompile("15;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `strconv.ParseInt: parsing "simuhuawei;1449573472;00002": invalid syntax` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Diff", Filters: []string{},
			Path: "Diff", Type: utils.MetaDifference,
			Value:     config.NewRSRParsersMustCompile("15;12;2", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := int64(1)
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, %T received: <%+v> %T", expected, expected, out, out)
	}
}

func TestAgReqParseFieldMetaMultiply(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Multiply", Filters: []string{},
			Path: "Multiply", Type: utils.MetaMultiply,
			Value:     config.NewRSRParsersMustCompile("15;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `strconv.ParseInt: parsing "simuhuawei;1449573472;00002": invalid syntax` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Multiply", Filters: []string{},
			Path: "Multiply", Type: utils.MetaMultiply,
			Value:     config.NewRSRParsersMustCompile("15;15", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := int64(225)
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, %T received: <%+v> %T", expected, expected, out, out)
	}
}

func TestAgReqParseFieldMetaDivide(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Divide", Filters: []string{},
			Path: "Divide", Type: utils.MetaDivide,
			Value:     config.NewRSRParsersMustCompile("15;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `strconv.ParseInt: parsing "simuhuawei;1449573472;00002": invalid syntax` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "Divide", Filters: []string{},
			Path: "Divide", Type: utils.MetaDivide,
			Value:     config.NewRSRParsersMustCompile("15;3", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := int64(5)
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, %T received: <%+v> %T", expected, expected, out, out)
	}
}

func TestAgReqParseFieldMetaValueExponent(t *testing.T) {
	//creater diameter message
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(2)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000004")), // Subscription-Id-Data
			diam.NewAVP(avp.ValueDigits, avp.Mbit, 0, datatype.Integer64(20000)),
		}})
	//create diameterDataProvider
	dP := newDADataProvider(nil, m)
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(dP, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "ValExp", Filters: []string{},
			Path: "ValExp", Type: utils.MetaValueExponent,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid arguments <[{"Rules":"~*req.Session-Id","AllFiltersMatch":true}]> to *value_exponent` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "ValExp", Filters: []string{},
			Path: "ValExp", Type: utils.MetaValueExponent,
			Value:     config.NewRSRParsersMustCompile("15;~*req.Session-Id", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `strconv.Atoi: parsing "simuhuawei;1449573472;00002": invalid syntax` {
		t.Error(err)
	}

	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "ValExp", Filters: []string{},
			Path: "ValExp", Type: utils.MetaValueExponent,
			Value:     config.NewRSRParsersMustCompile("~*req.Session-Id;15", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	if _, err := agReq.ParseField(tplFlds[0]); err == nil ||
		err.Error() != `invalid value <simuhuawei;1449573472;00002> to *value_exponent` {
		t.Error(err)
	}
	tplFlds = []*config.FCTemplate{
		&config.FCTemplate{Tag: "ValExp", Filters: []string{},
			Path: "ValExp", Type: utils.MetaValueExponent,
			Value:     config.NewRSRParsersMustCompile("2;3", true, utils.INFIELD_SEP),
			Mandatory: true},
	}
	expected := "2000"
	if out, err := agReq.ParseField(tplFlds[0]); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(out, expected) {
		t.Errorf("expecting: <%+v>, %T received: <%+v> %T", expected, expected, out, out)
	}
}

func TestAgReqOverwrite(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)
	agReq.CGRRequest.Set([]string{utils.AnswerTime},
		time.Date(2013, 12, 30, 15, 0, 1, 0, time.UTC), false, false)
	agReq.CGRRequest.Set([]string{utils.RequestType}, utils.META_PREPAID, false, false)

	agReq.CGRReply = config.NewNavigableMap(nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile(":", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("OverwrittenAccount", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.META_COMPOSED,
			Value: config.NewRSRParsersMustCompile("WithComposed", true, utils.INFIELD_SEP)},
	}

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	}

	if rcv, err := agReq.CGRReply.GetField([]string{utils.Account}); err != nil {
		t.Error(err)
	} else if sls, canCast := rcv.([]*config.NMItem); !canCast {
		t.Errorf("Cannot cast to []*config.NMItem %+v", rcv)
	} else if len(sls) != 1 {
		t.Errorf("expecting: %+v, \n received: %+v ", 1, len(sls))
	} else if sls[0].Data != "OverwrittenAccountWithComposed" {
		t.Errorf("expecting: %+v, \n received: %+v ",
			"OverwrittenAccountWithComposed", (rcv.([]*config.NMItem))[0].Data)
	}
}

func TestAgReqGroupType(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)
	agReq.CGRRequest.Set([]string{utils.AnswerTime},
		time.Date(2013, 12, 30, 15, 0, 1, 0, time.UTC), false, false)
	agReq.CGRRequest.Set([]string{utils.RequestType}, utils.META_PREPAID, false, false)

	agReq.CGRReply = config.NewNavigableMap(nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaGroup,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCgrep + utils.NestingSep + utils.Account, Type: utils.MetaGroup,
			Value: config.NewRSRParsersMustCompile("test", true, utils.INFIELD_SEP)},
	}

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	}

	if rcv, err := agReq.CGRReply.GetField([]string{utils.Account}); err != nil {
		t.Error(err)
	} else if sls, canCast := rcv.([]*config.NMItem); !canCast {
		t.Errorf("Cannot cast to []*config.NMItem %+v", rcv)
	} else if len(sls) != 2 {
		t.Errorf("expecting: %+v, \n received: %+v ", 1, len(sls))
	} else if sls[0].Data != "cgrates.org" {
		t.Errorf("expecting: %+v, \n received: %+v ", "cgrates.org", sls[0].Data)
	} else if sls[1].Data != "test" {
		t.Errorf("expecting: %+v, \n received: %+v ", "test", sls[0].Data)
	}
}

func TestAgReqSetFieldsInTmp(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaTmp + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaTmp + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.tmp, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.tmp)
	}
}

func TestAgReqSetFieldsWithRemove(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	// populate request, emulating the way will be done in HTTPAgent
	agReq.CGRRequest.Set([]string{utils.CGRID},
		utils.Sha1("dsafdsaf", time.Date(2013, 11, 7, 8, 42, 26, 0, time.UTC).String()),
		false, false)
	agReq.CGRRequest.Set([]string{utils.ToR}, utils.VOICE, false, false)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)
	agReq.CGRRequest.Set([]string{utils.Destination}, "1002", false, false)
	agReq.CGRRequest.Set([]string{utils.AnswerTime},
		time.Date(2013, 12, 30, 15, 0, 1, 0, time.UTC), false, false)
	agReq.CGRRequest.Set([]string{utils.RequestType}, utils.META_PREPAID, false, false)
	agReq.CGRRequest.Set([]string{utils.Usage}, time.Duration(3*time.Minute), false, false)

	cgrRply := map[string]interface{}{
		utils.CapAttributes: map[string]interface{}{
			"PaypalAccount": "cgrates@paypal.com",
		},
		utils.CapMaxUsage: time.Duration(120 * time.Second),
		utils.Error:       "",
	}
	agReq.CGRReply = config.NewNavigableMap(cgrRply)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaRep + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaRep + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Destination",
			Path: utils.MetaRep + utils.NestingSep + utils.Destination, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Destination", true, utils.INFIELD_SEP)},

		&config.FCTemplate{Tag: "RequestedUsageVoice",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*voice"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_seconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "RequestedUsageData",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*data"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_nanoseconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "RequestedUsageSMS",
			Path: utils.MetaRep + utils.NestingSep + "RequestedUsage", Type: utils.MetaVariable,
			Filters: []string{"*string:~*cgreq.ToR:*sms"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgreq.Usage{*duration_nanoseconds}", true, utils.INFIELD_SEP)},

		&config.FCTemplate{Tag: "AttrPaypalAccount",
			Path: utils.MetaRep + utils.NestingSep + "PaypalAccount", Type: utils.MetaVariable,
			Filters: []string{"*empty:~*cgrep.Error:"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.Attributes.PaypalAccount", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "MaxUsage",
			Path: utils.MetaRep + utils.NestingSep + "MaxUsage", Type: utils.MetaVariable,
			Filters: []string{"*empty:~*cgrep.Error:"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.MaxUsage{*duration_seconds}", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Error",
			Path: utils.MetaRep + utils.NestingSep + "Error", Type: utils.MetaVariable,
			Filters: []string{"*rsr::~*cgrep.Error(!^$)"},
			Value: config.NewRSRParsersMustCompile(
				"~*cgrep.Error", true, utils.INFIELD_SEP)},
	}
	eMp := config.NewNavigableMap(nil)
	eMp.Set([]string{utils.Tenant}, []*config.NMItem{
		&config.NMItem{Data: "cgrates.org", Path: []string{utils.Tenant},
			Config: tplFlds[0]}}, false, true)
	eMp.Set([]string{utils.Account}, []*config.NMItem{
		&config.NMItem{Data: "1001", Path: []string{utils.Account},
			Config: tplFlds[1]}}, false, true)
	eMp.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[2]}}, false, true)
	eMp.Set([]string{"RequestedUsage"}, []*config.NMItem{
		&config.NMItem{Data: "180", Path: []string{"RequestedUsage"},
			Config: tplFlds[3]}}, false, true)
	eMp.Set([]string{"PaypalAccount"}, []*config.NMItem{
		&config.NMItem{Data: "cgrates@paypal.com", Path: []string{"PaypalAccount"},
			Config: tplFlds[6]}}, false, true)
	eMp.Set([]string{"MaxUsage"}, []*config.NMItem{
		&config.NMItem{Data: "120", Path: []string{"MaxUsage"},
			Config: tplFlds[7]}}, false, true)

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.Reply, eMp) {
		t.Errorf("expecting: %+v,\n received: %+v", eMp, agReq.Reply)
	}

	tplFldsRemove := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaRep + utils.NestingSep + utils.Tenant, Type: utils.MetaRemove},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaRep + utils.NestingSep + utils.Account, Type: utils.MetaRemove},
	}

	eMpRemove := config.NewNavigableMap(nil)
	eMpRemove.Set([]string{utils.Destination}, []*config.NMItem{
		&config.NMItem{Data: "1002", Path: []string{utils.Destination},
			Config: tplFlds[2]}}, false, true)
	eMpRemove.Set([]string{"RequestedUsage"}, []*config.NMItem{
		&config.NMItem{Data: "180", Path: []string{"RequestedUsage"},
			Config: tplFlds[3]}}, false, true)
	eMpRemove.Set([]string{"PaypalAccount"}, []*config.NMItem{
		&config.NMItem{Data: "cgrates@paypal.com", Path: []string{"PaypalAccount"},
			Config: tplFlds[6]}}, false, true)
	eMpRemove.Set([]string{"MaxUsage"}, []*config.NMItem{
		&config.NMItem{Data: "120", Path: []string{"MaxUsage"},
			Config: tplFlds[7]}}, false, true)

	if err := agReq.SetFields(tplFldsRemove); err != nil {
		t.Error(err)
	} else if agReq.Reply.String() != eMpRemove.String() { // when compare ignore orders
		t.Errorf("expecting: %+v,\n received: %+v", eMpRemove.String(), agReq.Reply.String())
	}

	tplFldsRemove = []*config.FCTemplate{
		&config.FCTemplate{Path: utils.MetaRep, Type: utils.MetaRemoveAll},
	}

	eMpRemove = config.NewNavigableMap(nil)
	if err := agReq.SetFields(tplFldsRemove); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(agReq.Reply.Values(), eMpRemove.Values()) {
		t.Errorf("expecting: %+v,\n received: %+v", eMpRemove, agReq.Reply)
	}
}

func TestAgReqSetFieldsInCache(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	engine.InitCache(cfg.CacheCfg())
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaCache + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCache + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
	}

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	}

	if val, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Tenant}); err != nil {
		t.Error(err)
	} else if val.([]*config.NMItem)[0].Data != "cgrates.org" {
		t.Errorf("expecting: %+v, \n received: %+v ", "cgrates.org", utils.ToJSON(val))
	}

	if val, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Account}); err != nil {
		t.Error(err)
	} else if val.([]*config.NMItem)[0].Data != "1001" {
		t.Errorf("expecting: %+v, \n received: %+v ", "1001", utils.ToJSON(val))
	}

	if _, err := agReq.FieldAsInterface([]string{utils.MetaCache, "Unexist"}); err == nil || err != utils.ErrNotFound {
		t.Error(err)
	}
}

func TestAgReqSetFieldsInCacheWithTimeOut(t *testing.T) {
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)

	cfg.CacheCfg()[utils.CacheUCH].TTL = 1 * time.Second
	engine.InitCache(cfg.CacheCfg())
	agReq := NewAgentRequest(nil, nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)
	agReq.CGRRequest.Set([]string{utils.Account}, "1001", false, false)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Tenant",
			Path: utils.MetaCache + utils.NestingSep + utils.Tenant, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("cgrates.org", true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "Account",
			Path: utils.MetaCache + utils.NestingSep + utils.Account, Type: utils.MetaVariable,
			Value: config.NewRSRParsersMustCompile("~*cgreq.Account", true, utils.INFIELD_SEP)},
	}

	if err := agReq.SetFields(tplFlds); err != nil {
		t.Error(err)
	}

	if val, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Tenant}); err != nil {
		t.Error(err)
	} else if val.([]*config.NMItem)[0].Data != "cgrates.org" {
		t.Errorf("expecting: %+v, \n received: %+v ", "cgrates.org", utils.ToJSON(val))
	}

	if val, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Account}); err != nil {
		t.Error(err)
	} else if val.([]*config.NMItem)[0].Data != "1001" {
		t.Errorf("expecting: %+v, \n received: %+v ", "1001", utils.ToJSON(val))
	}

	if _, err := agReq.FieldAsInterface([]string{utils.MetaCache, "Unexist"}); err == nil || err != utils.ErrNotFound {
		t.Error(err)
	}
	// give enough time to Cache to remove ttl the *uch
	time.Sleep(2 * time.Second)
	if _, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Tenant}); err == nil || err != utils.ErrNotFound {
		t.Error(err)
	}
	if _, err := agReq.FieldAsInterface([]string{utils.MetaCache, utils.Account}); err == nil || err != utils.ErrNotFound {
		t.Error(err)
	}
}

func TestAgReqFiltersInsideField(t *testing.T) {
	//simulate the diameter request
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String("bb97be2b9f37c2be9614fff71c8b1d08b1acbff8"))
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("192.168.1.1"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	m.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Enumerated(3))
	m.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(2))
	m.NewAVP(avp.DestinationHost, avp.Mbit, 0, datatype.DiameterIdentity("CGR-DA"))
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String("voice@DiamItCCRInit"))
	m.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Date(2018, 10, 4, 15, 12, 20, 0, time.UTC)))
	m.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),      // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("1006")), // Subscription-Id-Data
		}})
	m.NewAVP(avp.ServiceIdentifier, avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(avp.UsedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(250))}})
	m.NewAVP(873, avp.Mbit, 10415, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(20300, avp.Mbit, 2011, &diam.GroupedAVP{ // IN-Information
				AVP: []*diam.AVP{
					diam.NewAVP(831, avp.Mbit, 10415, datatype.UTF8String("1006")),                                      // Calling-Party-Address
					diam.NewAVP(832, avp.Mbit, 10415, datatype.UTF8String("1002")),                                      // Called-Party-Address
					diam.NewAVP(20327, avp.Mbit, 2011, datatype.UTF8String("1002")),                                     // Real-Called-Number
					diam.NewAVP(20339, avp.Mbit, 2011, datatype.Unsigned32(0)),                                          // Charge-Flow-Type
					diam.NewAVP(20302, avp.Mbit, 2011, datatype.UTF8String("")),                                         // Calling-Vlr-Number
					diam.NewAVP(20303, avp.Mbit, 2011, datatype.UTF8String("")),                                         // Calling-CellID-Or-SAI
					diam.NewAVP(20313, avp.Mbit, 2011, datatype.OctetString("")),                                        // Bearer-Capability
					diam.NewAVP(20321, avp.Mbit, 2011, datatype.UTF8String("bb97be2b9f37c2be9614fff71c8b1d08b1acbff8")), // Call-Reference-Number
					diam.NewAVP(20322, avp.Mbit, 2011, datatype.UTF8String("")),                                         // MSC-Address
					diam.NewAVP(20324, avp.Mbit, 2011, datatype.Unsigned32(0)),                                          // Time-Zone
					diam.NewAVP(20385, avp.Mbit, 2011, datatype.UTF8String("")),                                         // Called-Party-NP
					diam.NewAVP(20386, avp.Mbit, 2011, datatype.UTF8String("")),                                         // SSP-Time
				},
			}),
		}})
	//create diameterDataProvider
	cfg, _ := config.NewDefaultCGRConfig()
	data := engine.NewInternalDB(nil, nil, true, cfg.DataDbCfg().Items)
	dm := engine.NewDataManager(data, config.CgrConfig().CacheCfg(), nil)
	filterS := engine.NewFilterS(cfg, nil, dm)
	//pass the data provider to agent request
	agReq := NewAgentRequest(newDADataProvider(nil, m), nil, nil, nil, nil, "cgrates.org", "", filterS, nil, nil)

	tplFlds := []*config.FCTemplate{
		&config.FCTemplate{Tag: "Usage",
			Path: utils.MetaCgreq + utils.NestingSep + utils.Usage, Type: utils.MetaCCUsage,
			Value: config.NewRSRParsersMustCompile("~*req.CC-Request-Number;~*req.Used-Service-Unit.CC-Time:s/(.*)/${1}s/;5m",
				true, utils.INFIELD_SEP)},
		&config.FCTemplate{Tag: "AnswerTime",
			Path: utils.MetaCgreq + utils.NestingSep + utils.AnswerTime, Type: utils.MetaDifference,
			Filters: []string{"*gt:~*cgreq.Usage:0s"}, // populate answer time if usage is greater than zero
			Value:   config.NewRSRParsersMustCompile("~*req.Event-Timestamp;~*cgreq.Usage", true, utils.INFIELD_SEP)},
	}

	if err := agReq.SetFields(tplFlds); err != nil {
		// here we get error
		//t.Error(err)
	}

}
