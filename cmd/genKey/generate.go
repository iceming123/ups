package main

import (
	"encoding/hex"
	"fmt"

	"github.com/iceming123/ups/cmd/utils"
	"github.com/iceming123/ups/crypto"
	"gopkg.in/urfave/cli.v1"
)

var commandGenerate = cli.Command{
	Name:      "generate",
	Usage:     "generate new key item",
	ArgsUsage: "",
	Description: `
Generate a new key item.
`,
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "sum",
			Usage: "key info count",
			Value: 1,
		},
	},
	Action: func(ctx *cli.Context) error {
		count := ctx.GlobalInt("sum")
		if count <= 0 || count > 100 {
			count = 100
		}

		for i := 0; i < count; i++ {
			if priv, err := crypto.GenerateKey(); err != nil {
				utils.Fatalf("Error GenerateKey: %v", err)
			} else {
				fmt.Println("privkey:", hex.EncodeToString(crypto.FromECDSA(priv)))
				fmt.Println("pubkey:", hex.EncodeToString(crypto.FromECDSAPub(&priv.PublicKey)))
				addr := crypto.PubkeyToAddress(priv.PublicKey)
				fmt.Println("address:", crypto.AddressToHex(addr))
				fmt.Println("-------------------------------------------------------")
			}
		}
		return nil
	},
}
