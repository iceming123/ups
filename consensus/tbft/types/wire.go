package types

import (
	"github.com/tendermint/go-amino"
	"github.com/iceming123/ups/consensus/tbft/crypto/cryptoamino"
)

var cdc = amino.NewCodec()

func init() {
	RegisterBlockAmino(cdc)
}

//RegisterBlockAmino is register for block amino
func RegisterBlockAmino(cdc *amino.Codec) {
	cryptoAmino.RegisterAmino(cdc)
}
