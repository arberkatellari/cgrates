/*
Real-time Online/Offline Charging System (OerS) for Telecom & ISP environments
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

package accounts

import (
	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/ericlagergren/decimal"
)

// cloneUnitsFromConcretes returns cloned units from the concrete balances passed as parameters
func cloneUnitsFromConcretes(cBs []*concreteBalance) (clnedUnts []*utils.Decimal) {
	if cBs == nil {
		return
	}
	clnedUnts = make([]*utils.Decimal, len(cBs))
	for i := range cBs {
		clnedUnts[i] = cBs[i].blnCfg.Units.Clone()
	}
	return
}

// restoreUnitsFromClones will restore the units from the clones
func restoreUnitsFromClones(cBs []*concreteBalance, clnedUnts []*utils.Decimal) {
	for i, clnedUnt := range clnedUnts {
		cBs[i].blnCfg.Units.Big = clnedUnt.Big
	}
}

// newConcreteBalance constructs a concreteBalanceOperator
func newConcreteBalanceOperator(acntID string, blnCfg *utils.Balance,
	fltrS *engine.FilterS, connMgr *engine.ConnManager,
	attrSConns, rateSConns []string) balanceOperator {
	return &concreteBalance{acntID, blnCfg, fltrS, connMgr, attrSConns, rateSConns}
}

// concreteBalance is the operator for *concrete balance type
type concreteBalance struct {
	acntID     string
	blnCfg     *utils.Balance
	fltrS      *engine.FilterS
	connMgr    *engine.ConnManager
	attrSConns []string
	rateSConns []string
}

// id implements the balanceOperator interface
func (cB *concreteBalance) id() string {
	return cB.blnCfg.ID
}

// debitAbstracts implements the balanceOperator interface
func (cB *concreteBalance) debitAbstracts(aUnits *decimal.Big,
	cgrEv *utils.CGREvent, dbted *decimal.Big) (ec *utils.EventCharges, err error) {
	evNm := cgrEv.AsDataProvider()
	// pass the general balance filters
	var pass bool
	if pass, err = cB.fltrS.Pass(context.TODO(), cgrEv.Tenant, cB.blnCfg.FilterIDs, evNm); err != nil {
		return
	} else if !pass {
		return nil, utils.ErrFilterNotPassingNoCaps
	}

	// costIncrement
	var costIcrm *utils.CostIncrement
	if costIcrm, err = costIncrement(cB.blnCfg.CostIncrements,
		cB.fltrS, cgrEv.Tenant, evNm); err != nil {
		return
	}
	var ecCncrt *utils.EventCharges
	if ecCncrt, err = maxDebitAbstractsFromConcretes(aUnits,
		cB.acntID, []*concreteBalance{cB},
		cB.connMgr, cgrEv,
		cB.attrSConns, cB.blnCfg.AttributeIDs,
		cB.rateSConns, cB.blnCfg.RateProfileIDs,
		costIcrm, dbted); err != nil {
		return
	} else if ecCncrt.Abstracts.Compare(utils.NewDecimal(0, 0)) == 0 { // no debit performed
		return
	}
	ec = utils.NewEventCharges()
	ec.Abstracts = ecCncrt.Abstracts
	ec.Concretes = ecCncrt.Concretes
	// RatingID
	var ratingID string
	if costIcrm != nil {
		ratingID = utils.UUIDSha1Prefix()
		ec.Rating[ratingID] = &utils.RateSInterval{
			Increments: []*utils.RateSIncrement{
				{
					/*
						Rate: &utils.Rate{
							ID: utils.MetaCostIncrement,
							IntervalRates: []*utils.IntervalRate{
								{
									FixedFee:     costIcrm.FixedFee,
									RecurrentFee: costIcrm.RecurrentFee,
								},
							},
						},

					*/

					CompressFactor: 1,
				},
			},
			CompressFactor: 1,
		}
	} else { // take it from first increment
		ratingID = ecCncrt.Accounting[ecCncrt.Charges[0].ChargingID].RatingID
		ec.Rating[ratingID] = ecCncrt.Rating[ratingID]
	}
	// AccountingID
	acntID := utils.UUIDSha1Prefix()
	ec.Accounting[acntID] = &utils.AccountCharge{
		AccountID: cB.acntID,
		BalanceID: utils.MetaTransAbstract,
		Units:     ec.Abstracts,
		RatingID:  ratingID,
	}
	for _, cE := range ecCncrt.Charges {
		if ecCncrt.Accounting[cE.ChargingID].UnitFactorID != utils.EmptyString {
			ec.UnitFactors[ecCncrt.Accounting[cE.ChargingID].UnitFactorID] = ecCncrt.UnitFactors[ecCncrt.Accounting[cE.ChargingID].UnitFactorID]
		}
		ec.Accounting[cE.ChargingID] = ecCncrt.Accounting[cE.ChargingID]
		ec.Accounting[cE.ChargingID].RatingID = utils.EmptyString // should not be needed since we have used it above
		ec.Accounting[acntID].JoinedChargeIDs = append(ec.Accounting[acntID].JoinedChargeIDs, cE.ChargingID)
	}
	ec.Charges = []*utils.ChargeEntry{
		{
			ChargingID:     acntID,
			CompressFactor: 1,
		},
	}
	return
}

// debitConcretes implements the balanceOperator interface
func (cB *concreteBalance) debitConcretes(cUnits *decimal.Big,
	cgrEv *utils.CGREvent, debited *decimal.Big) (ec *utils.EventCharges, err error) {
	evNm := cgrEv.AsDataProvider()
	// pass the general balance filters
	var pass bool
	if pass, err = cB.fltrS.Pass(context.TODO(), cgrEv.Tenant, cB.blnCfg.FilterIDs, evNm); err != nil {
		return
	} else if !pass {
		return nil, utils.ErrFilterNotPassingNoCaps
	}

	// unitFactor
	var uF *utils.UnitFactor
	if uF, err = unitFactor(cB.blnCfg.UnitFactors, cB.fltrS, cgrEv.Tenant, evNm); err != nil {
		return
	}
	var hasUF bool
	if uF != nil && uF.Factor.Cmp(decimal.New(1, 0)) != 0 {
		hasUF = true
		cUnits = utils.MultiplyBig(cUnits, uF.Factor.Big)
	}

	// balanceLimit
	var hasLmt bool
	var blncLmt *utils.Decimal
	if blncLmt, err = balanceLimit(cB.blnCfg.Opts); err != nil {
		return
	}
	if blncLmt != nil && blncLmt.Big.Cmp(decimal.New(0, 0)) != 0 {
		cB.blnCfg.Units.Big = utils.SubstractBig(cB.blnCfg.Units.Big, blncLmt.Big)
		hasLmt = true
	}
	var dbted *decimal.Big
	if cB.blnCfg.Units.Big.Cmp(cUnits) <= 0 && blncLmt != nil { // balance smaller than debit and limited
		dbted = cB.blnCfg.Units.Big
		cB.blnCfg.Units.Big = blncLmt.Big
	} else {
		cB.blnCfg.Units.Big = utils.SubstractBig(cB.blnCfg.Units.Big, cUnits)
		if hasLmt { // put back the limit
			cB.blnCfg.Units.Big = utils.SumBig(cB.blnCfg.Units.Big, blncLmt.Big)
		}
		dbted = cUnits
	}
	if hasUF {
		dbted = utils.DivideBig(dbted, uF.Factor.Big)
	}
	if dbted.Cmp(decimal.New(0, 0)) == 0 {
		return // no event cost for 0 debit
	}
	// EventCharges
	ec = utils.NewEventCharges()
	ec.Concretes = &utils.Decimal{dbted}
	// UnitFactors
	var ufID string
	if hasUF {
		ufID = utils.UUIDSha1Prefix()
		ec.UnitFactors[ufID] = uF
	}
	acntID := utils.UUIDSha1Prefix()
	ec.Accounting[acntID] = &utils.AccountCharge{
		AccountID:    cB.acntID,
		BalanceID:    cB.blnCfg.ID,
		Units:        &utils.Decimal{dbted},
		BalanceLimit: blncLmt,
		UnitFactorID: ufID,
	}
	ec.Charges = []*utils.ChargeEntry{
		{
			ChargingID:     acntID,
			CompressFactor: 1,
		},
	}

	return
}