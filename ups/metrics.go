// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ups

import (
	"github.com/iceming123/ups/metrics"
	"github.com/iceming123/ups/p2p"
)

var (
	propTxnInPacketsMeter     = metrics.NewRegisteredMeter("ups/prop/txns/in/packets", nil)
	propTxnInTxsMeter         = metrics.NewRegisteredMeter("ups/prop/txns/in/txs", nil)
	propTxnInTrafficMeter     = metrics.NewRegisteredMeter("ups/prop/txns/in/traffic", nil)
	propTxnOutPacketsMeter    = metrics.NewRegisteredMeter("ups/prop/txns/out/packets", nil)
	propTxnOutTrafficMeter    = metrics.NewRegisteredMeter("ups/prop/txns/out/traffic", nil)
	propFtnInPacketsMeter     = metrics.NewRegisteredMeter("ups/prop/ftns/in/packets", nil)
	propFtnInTrafficMeter     = metrics.NewRegisteredMeter("ups/prop/ftns/in/traffic", nil)
	propFtnOutPacketsMeter    = metrics.NewRegisteredMeter("ups/prop/ftns/out/packets", nil)
	propFtnOutTrafficMeter    = metrics.NewRegisteredMeter("ups/prop/ftns/out/traffic", nil)
	propFHashInPacketsMeter   = metrics.NewRegisteredMeter("ups/prop/fhashes/in/packets", nil)
	propFHashInTrafficMeter   = metrics.NewRegisteredMeter("ups/prop/fhashes/in/traffic", nil)
	propFHashOutPacketsMeter  = metrics.NewRegisteredMeter("ups/prop/fhashes/out/packets", nil)
	propFHashOutTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/fhashes/out/traffic", nil)
	propSHashInPacketsMeter   = metrics.NewRegisteredMeter("ups/prop/shashes/in/packets", nil)
	propSHashInTrafficMeter   = metrics.NewRegisteredMeter("ups/prop/shashes/in/traffic", nil)
	propSHashOutPacketsMeter  = metrics.NewRegisteredMeter("ups/prop/shashes/out/packets", nil)
	propSHashOutTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/shashes/out/traffic", nil)
	propFBlockInPacketsMeter  = metrics.NewRegisteredMeter("ups/prop/fblocks/in/packets", nil)
	propFBlockInTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/fblocks/in/traffic", nil)
	propFBlockOutPacketsMeter = metrics.NewRegisteredMeter("ups/prop/fblocks/out/packets", nil)
	propFBlockOutTrafficMeter = metrics.NewRegisteredMeter("ups/prop/fblocks/out/traffic", nil)
	propSBlockInPacketsMeter  = metrics.NewRegisteredMeter("ups/prop/sblocks/in/packets", nil)
	propSBlockInTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/sblocks/in/traffic", nil)
	propSBlockOutPacketsMeter = metrics.NewRegisteredMeter("ups/prop/sblocks/out/packets", nil)
	propSBlockOutTrafficMeter = metrics.NewRegisteredMeter("ups/prop/sblocks/out/traffic", nil)

	propNodeInfoInPacketsMeter    = metrics.NewRegisteredMeter("ups/prop/nodeinfo/in/packets", nil)
	propNodeInfoInTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/nodeinfo/in/traffic", nil)
	propNodeInfoOutPacketsMeter = metrics.NewRegisteredMeter("ups/prop/nodeinfo/out/packets", nil)
	propNodeInfoOutTrafficMeter = metrics.NewRegisteredMeter("ups/prop/nodeinfo/out/traffic", nil)

	propNodeInfoHashInPacketsMeter    = metrics.NewRegisteredMeter("ups/prop/nodeinfohash/in/packets", nil)
	propNodeInfoHashInTrafficMeter  = metrics.NewRegisteredMeter("ups/prop/nodeinfohash/in/traffic", nil)
	propNodeInfoHashOutPacketsMeter = metrics.NewRegisteredMeter("ups/prop/nodeinfohash/out/packets", nil)
	propNodeInfoHashOutTrafficMeter = metrics.NewRegisteredMeter("ups/prop/nodeinfohash/out/traffic", nil)



	reqFHeaderInPacketsMeter  = metrics.NewRegisteredMeter("ups/req/headers/in/packets", nil)
	reqFHeaderInTrafficMeter  = metrics.NewRegisteredMeter("ups/req/headers/in/traffic", nil)
	reqFHeaderOutPacketsMeter = metrics.NewRegisteredMeter("ups/req/headers/out/packets", nil)
	reqFHeaderOutTrafficMeter = metrics.NewRegisteredMeter("ups/req/headers/out/traffic", nil)

	reqFBodyInPacketsMeter  = metrics.NewRegisteredMeter("ups/req/fbodies/in/packets", nil)
	reqFBodyInTrafficMeter  = metrics.NewRegisteredMeter("ups/req/fbodies/in/traffic", nil)
	reqFBodyOutPacketsMeter = metrics.NewRegisteredMeter("ups/req/fbodies/out/packets", nil)
	reqFBodyOutTrafficMeter = metrics.NewRegisteredMeter("ups/req/fbodies/out/traffic", nil)

	reqStateInPacketsMeter    = metrics.NewRegisteredMeter("ups/req/states/in/packets", nil)
	reqStateInTrafficMeter    = metrics.NewRegisteredMeter("ups/req/states/in/traffic", nil)
	reqStateOutPacketsMeter   = metrics.NewRegisteredMeter("ups/req/states/out/packets", nil)
	reqStateOutTrafficMeter   = metrics.NewRegisteredMeter("ups/req/states/out/traffic", nil)
	reqReceiptInPacketsMeter  = metrics.NewRegisteredMeter("ups/req/receipts/in/packets", nil)
	reqReceiptInTrafficMeter  = metrics.NewRegisteredMeter("ups/req/receipts/in/traffic", nil)
	reqReceiptOutPacketsMeter = metrics.NewRegisteredMeter("ups/req/receipts/out/packets", nil)
	reqReceiptOutTrafficMeter = metrics.NewRegisteredMeter("ups/req/receipts/out/traffic", nil)

	getHeadInPacketsMeter  = metrics.NewRegisteredMeter("ups/get/head/in/packets", nil)
	getHeadInTrafficMeter  = metrics.NewRegisteredMeter("ups/get/head/in/traffic", nil)
	getHeadOutPacketsMeter = metrics.NewRegisteredMeter("ups/get/head/out/packets", nil)
	getHeadOutTrafficMeter = metrics.NewRegisteredMeter("ups/get/head/out/traffic", nil)

	getNodeInfoInPacketsMeter    = metrics.NewRegisteredMeter("ups/get/nodeinfo/in/packets", nil)
	getNodeInfoInTrafficMeter  = metrics.NewRegisteredMeter("ups/get/nodeinfo/in/traffic", nil)
	getNodeInfoOutPacketsMeter = metrics.NewRegisteredMeter("ups/get/nodeinfo/out/packets", nil)
	getNodeInfoOutTrafficMeter = metrics.NewRegisteredMeter("ups/get/nodeinfo/out/traffic", nil)

	miscInPacketsMeter  = metrics.NewRegisteredMeter("ups/misc/in/packets", nil)
	miscInTrafficMeter  = metrics.NewRegisteredMeter("ups/misc/in/traffic", nil)
	miscOutPacketsMeter = metrics.NewRegisteredMeter("ups/misc/out/packets", nil)
	miscOutTrafficMeter = metrics.NewRegisteredMeter("ups/misc/out/traffic", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqFHeaderInPacketsMeter, reqFHeaderInTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqFBodyInPacketsMeter, reqFBodyInTrafficMeter
	case msg.Code == NodeDataMsg:
		packets, traffic = reqStateInPacketsMeter, reqStateInTrafficMeter
	case msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptInPacketsMeter, reqReceiptInTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propFHashInPacketsMeter, propFHashInTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propFBlockInPacketsMeter, propFBlockInTrafficMeter
	case msg.Code == TransactionMsg:
		packets, traffic = propTxnInPacketsMeter, propTxnInTrafficMeter
	case msg.Code == TbftNodeInfoMsg:
		packets, traffic = propNodeInfoInPacketsMeter, propNodeInfoInTrafficMeter
	case msg.Code == TbftNodeInfoHashMsg:
		packets, traffic = propNodeInfoHashInPacketsMeter, propNodeInfoHashInTrafficMeter
	case msg.Code == GetTbftNodeInfoMsg:
		packets, traffic = getNodeInfoInPacketsMeter, getNodeInfoInTrafficMeter
	case msg.Code == GetBlockHeadersMsg:
		packets, traffic = getHeadInPacketsMeter, getHeadInTrafficMeter
	case msg.Code == GetBlockBodiesMsg:
		packets, traffic = getHeadInPacketsMeter, getHeadInTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))
	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	switch {
	case msg.Code == BlockHeadersMsg:
		packets, traffic = reqFHeaderOutPacketsMeter, reqFHeaderOutTrafficMeter
	case msg.Code == BlockBodiesMsg:
		packets, traffic = reqFBodyOutPacketsMeter, reqFBodyOutTrafficMeter
	case msg.Code == NodeDataMsg:
		packets, traffic = reqStateOutPacketsMeter, reqStateOutTrafficMeter
	case msg.Code == ReceiptsMsg:
		packets, traffic = reqReceiptOutPacketsMeter, reqReceiptOutTrafficMeter

	case msg.Code == NewBlockHashesMsg:
		packets, traffic = propFHashOutPacketsMeter, propFHashOutTrafficMeter
	case msg.Code == NewBlockMsg:
		packets, traffic = propFBlockOutPacketsMeter, propFBlockOutTrafficMeter
	case msg.Code == TransactionMsg:
		packets, traffic = propTxnOutPacketsMeter, propTxnOutTrafficMeter
	case msg.Code == TbftNodeInfoMsg:
		packets, traffic = propNodeInfoOutPacketsMeter, propNodeInfoOutTrafficMeter
	case msg.Code == TbftNodeInfoHashMsg:
		packets, traffic = propNodeInfoHashOutPacketsMeter, propNodeInfoHashOutTrafficMeter
	case msg.Code == GetTbftNodeInfoMsg:
		packets, traffic = getNodeInfoOutPacketsMeter, getNodeInfoOutTrafficMeter
	case msg.Code == GetBlockHeadersMsg:
		packets, traffic = getHeadOutPacketsMeter, getHeadOutTrafficMeter
	case msg.Code == GetBlockBodiesMsg:
		packets, traffic = getHeadInPacketsMeter, getHeadOutTrafficMeter
	}
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
