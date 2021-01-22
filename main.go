package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto"
	keys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	swaptypes "github.com/tendermint/liquidity/x/liquidity/types"
	"google.golang.org/grpc"
)

var grpcConn *grpc.ClientConn

func signtxsend(round int, txnum int, msgnum int, msg *swaptypes.MsgSwap, priv cryptotypes.PrivKey, address sdk.AccAddress, w *sync.WaitGroup) {
	defer w.Done()
	for i := 0; i < round; i++ {
		startTime := time.Now()
		var txBytes [][]byte
		var msgs []sdk.Msg
		accSeq, accNum := accountinfo(address)
		for j := 0; j < msgnum; j++ {
			msgs = append(msgs, msg)
		}
		encCfg := simapp.MakeTestEncodingConfig()
		txBuilder := encCfg.TxConfig.NewTxBuilder()
		err := txBuilder.SetMsgs(msgs...)
		if err != nil {
			println(err)
		}
		txBuilder.SetGasLimit(150000)
		txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(150))))
		for k := 0; k < txnum; k++ {
			sigV2 := signing.SignatureV2{
				PubKey: priv.PubKey(),
				Data: &signing.SingleSignatureData{
					SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
					Signature: nil,
				},
				Sequence: accSeq,
			}

			err = txBuilder.SetSignatures(sigV2)
			if err != nil {
				println(err)
			}

			// Second round: all signer infos are set, so each signer can sign.
			signerData := xauthsigning.SignerData{
				ChainID:       "swap-testnet-2001",
				AccountNumber: accNum,
				Sequence:      accSeq,
			}
			sigV2, err = clienttx.SignWithPrivKey(
				encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData,
				txBuilder, priv, encCfg.TxConfig, accSeq)
			accSeq = accSeq + 1
			if err != nil {
				println(err)
			}

			err = txBuilder.SetSignatures(sigV2)
			if err != nil {
				println(err)
			}
			txByte, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
			if err != nil {
				println(err)
			}
			txBytes = append(txBytes, txByte)
		}
		for _, txByte := range txBytes {
			sendtx(txByte)
		}
		fmt.Printf("%d round end - ", i+1)
		fmt.Printf("account:%s", address.String())
		fmt.Printf(" Tx %d send!! ", txnum)
		elapsedTime := time.Since(startTime)
		fmt.Printf("TIME: %s\n", elapsedTime)
	}
}

func sendtx(txBytes []byte) {
	if grpcConn == nil {
		grpcclient()
	}
	txClient := tx.NewServiceClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	txClient.BroadcastTx(
		ctx,
		&tx.BroadcastTxRequest{
			Mode:    tx.BroadcastMode_BROADCAST_MODE_ASYNC,
			TxBytes: txBytes, // Proto-binary of the signed transaction, see previous step.
		},
	)
}

func accountinfo(addr sdk.AccAddress) (uint64, uint64) {
	if grpcConn == nil {
		grpcclient()
	}
	var acc authtypes.BaseAccount
	authClient := authtypes.NewQueryClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	authRes, err := authClient.Account(
		ctx,
		&authtypes.QueryAccountRequest{Address: addr.String()},
	)
	err = acc.Unmarshal(authRes.Account.Value)
	if err != nil {
		println(err)
	}
	return acc.Sequence, acc.AccountNumber
}

func grpcclient() {
	connV, err := grpc.Dial("localhost:9090", grpc.WithInsecure(), grpc.WithBlock())
	grpcConn = connV
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
}

func main() {
	var txnum int = 500
	var msgnum int = 1
	var round int = 10

	keyring, err := keys.New("swapchain", "os", "/root/.liquidityd/", nil)
	if err != nil {
		log.Fatalf("did not keyring: %v", err)
	}

	keylist, _ := keyring.List()

	wait := new(sync.WaitGroup)
	wait.Add(len(keylist))

	for _, key := range keylist {

		accountarmor, err := keyring.ExportPrivKeyArmor(key.GetName(), "qwer1234")
		if err != nil {
			println(err)
		}
		accountpriv, _, err := crypto.UnarmorDecryptPrivKey(accountarmor, "qwer1234")
		if err != nil {
			println(err)
		}
		swapcoin := types.NewInt64Coin("uatom", 1000000)
		orderpirce, _ := types.NewDecFromStr("0.148648648648648657")
		msg := swaptypes.NewMsgSwap(key.GetAddress(), 10, 1, swapcoin, "uusdt", orderpirce)
		go signtxsend(round, txnum, msgnum, msg, accountpriv, key.GetAddress(), wait)

	}

	wait.Wait()

}
