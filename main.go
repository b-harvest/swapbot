package main

import (
	"context"
	"fmt"
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

func signtx(msgnum int, msg *swaptypes.MsgSwap, priv cryptotypes.PrivKey, accSeq uint64, accNum uint64) []byte {
	var msgs []sdk.Msg
	for i := 0; i < msgnum; i++ {
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
	if err != nil {
		println(err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		println(err)
	}
	txBytes, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		println(err)
	}
	return txBytes
}

func sendtx(grpcConn *grpc.ClientConn, txBytes []byte) {
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

func accountinfo(addr sdk.AccAddress, grpcConn *grpc.ClientConn) (uint64, uint64) {
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
func main() {

	var txBytes [][]byte
	var txnum int = 100
	var msgnum int = 50

	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",    // your gRPC server address.
		grpc.WithInsecure(), // The SDK doesn't support any transport security mechanism.
	)
	if err != nil {
		println(err)
	}
	defer grpcConn.Close()
	keyring, err := keys.New("swapchain", "os", "/root/.liquidityd/", nil)
	keylist, _ := keyring.List()
	//accounttest, err := keyring.Key("validator")
	startTime := time.Now()
	for _, key := range keylist {

		accountarmor, err := keyring.ExportPrivKeyArmor(key.GetName(), "qwer1234")
		if err != nil {
			println(err)
		}
		accountpriv, _, err := crypto.UnarmorDecryptPrivKey(accountarmor, "qwer1234")
		if err != nil {
			println(err)
		}
		seq, accnum := accountinfo(key.GetAddress(), grpcConn)
		println(seq, accnum)
		swapcoin := types.NewInt64Coin("uatom", 1000000)
		orderpirce := sdk.NewDec(148648648648648657)
		msg := swaptypes.NewMsgSwap(key.GetAddress(), 10, 1, swapcoin, "uusdt", orderpirce)
		for i := 0; i < txnum; i++ {
			txByte := signtx(msgnum, msg, accountpriv, seq, accnum)
			txBytes = append(txBytes, txByte)
			seq = seq + 1
		}
	}
	var count int = 1
	for _, txByte := range txBytes {
		sendtx(grpcConn, txByte)
		println(count)
		count = count + 1
	}
	elapsedTime := time.Since(startTime)

	fmt.Printf("TIME: %s\n", elapsedTime)

}
