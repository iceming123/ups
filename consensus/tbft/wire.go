package tbft

import (
	"github.com/tendermint/go-amino"
	"github.com/iceming123/ups/consensus/tbft/types"
)

var cdc = amino.NewCodec()

func init() {
	RegisterConsensusMessages(cdc)
	// RegisterWALMessages(cdc)
	types.RegisterBlockAmino(cdc)
}
