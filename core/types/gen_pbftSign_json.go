// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"math/big"

	"github.com/iceming123/ups/common"
	"github.com/iceming123/ups/common/hexutil"
)

var _ = (*pbftSignMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (p PbftSign) MarshalJSON() ([]byte, error) {
	type PbftSign struct {
		FastHeight *hexutil.Big
		FastHash   common.Hash
		Result     uint32
		Sign       hexutil.Bytes
	}
	var enc PbftSign
	enc.FastHeight = (*hexutil.Big)(p.FastHeight)
	enc.FastHash = p.FastHash
	enc.Result = p.Result
	enc.Sign = p.Sign
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (p *PbftSign) UnmarshalJSON(input []byte) error {
	type PbftSign struct {
		FastHeight *hexutil.Big
		FastHash   *common.Hash
		Result     *uint32
		Sign       *hexutil.Bytes
	}
	var dec PbftSign
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.FastHeight != nil {
		p.FastHeight = (*big.Int)(dec.FastHeight)
	}
	if dec.FastHash != nil {
		p.FastHash = *dec.FastHash
	}
	if dec.Result != nil {
		p.Result = *dec.Result
	}
	if dec.Sign != nil {
		p.Sign = *dec.Sign
	}
	return nil
}
