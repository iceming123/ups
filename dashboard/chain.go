// Copyright 2018 The UpsChain Authors
// This file is part of the ups library.
//
// The ups library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ups library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ups library. If not, see <http://www.gnu.org/licenses/>.

package dashboard

import (
	"time"
)

// fastChainInfo contains the info of fast chain.
type fastChainInfo struct {
	LastFastTime ChartEntries `json:"lastFastTime,omitempty"`
	LastTxsCount ChartEntries `json:"lastTxsCount,omitempty"`
	GasSpending  ChartEntries `json:"gasSpending,omitempty"`
	GasLimit     ChartEntries `json:"gasLimit,omitempty"`
}
// collectTxpoolData gathers data about the tx_pool and sends it to the clients.
func (db *Dashboard) collectChainData() {
	defer db.wg.Done()
	fastchain := db.ups.BlockChain()

	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			lastFastTime := fastchain.CurrentHeader().Time
			lastTxsCount := len(fastchain.CurrentBlock().Body().Transactions)
			gasSpending := fastchain.CurrentBlock().GasUsed()
			gasLimit := fastchain.CurrentBlock().GasLimit()
			fastTime := &ChartEntry{
				Value: float64(lastFastTime.Uint64()),
			}
			txsCount := &ChartEntry{
				Value: float64(lastTxsCount),
			}
			spending := &ChartEntry{
				Value: float64(gasSpending),
			}
			limit := &ChartEntry{
				Value: float64(gasLimit),
			}
			fastChainInfo := &fastChainInfo{
				LastFastTime: append([]*ChartEntry{}, fastTime),
				LastTxsCount: append([]*ChartEntry{}, txsCount),
				GasSpending:  append([]*ChartEntry{}, spending),
				GasLimit:     append([]*ChartEntry{}, limit),
			}

			db.chainLock.Lock()
			db.history.Chain = &ChainMessage{
				FastChain:  fastChainInfo,
			}
			db.chainLock.Unlock()

			db.sendToAll(&Message{
				Chain: &ChainMessage{
					FastChain:  fastChainInfo,
				},
			})
		}
	}
}
