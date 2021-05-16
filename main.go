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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibctypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channelutils "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/client/utils"
	"google.golang.org/grpc"
)

var grpcConn *grpc.ClientConn

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

func signtxsend(round int, txnum int, msgnum int, priv cryptotypes.PrivKey, w *sync.WaitGroup, srcPort string, srcChannel string, coin sdk.Coin, sender sdk.AccAddress, receiver string) {
	defer w.Done()
	for i := 0; i < round; i++ {
		startTime := time.Now()
		var txBytes [][]byte

		accSeq, accNum := accountinfo(sender)

		msgs := msgcreationbot(msgnum, srcPort, srcChannel, coin, sender, receiver)
		encCfg := simapp.MakeTestEncodingConfig()
		txBuilder := encCfg.TxConfig.NewTxBuilder()
		err := txBuilder.SetMsgs(msgs...)
		if err != nil {
			println(err)
		}
		txBuilder.SetGasLimit(8999999999999999999)
		txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(0))))
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
				ChainID:       "swap-testnet-2004",
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
		fmt.Printf("sender account:%s", sender.String())
		fmt.Printf("receiver account:%s", receiver)
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

func msgcreationbot(msgnum int, srcPort string, srcChannel string, coin sdk.Coin, sender sdk.AccAddress, receiver string) []sdk.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	consensusState, height, _, err := channelutils.QueryLatestConsensusState(ctx, srcPort, srcChannel)
	if err != nil {
		println(err)
	}
	var timeoutHeight clienttypes.Height
	var timeoutTimestamp uint64

	absoluteHeight := height
	absoluteHeight.RevisionNumber += timeoutHeight.RevisionNumber
	absoluteHeight.RevisionHeight += timeoutHeight.RevisionHeight
	timeoutHeight = absoluteHeight

	timeoutTimestamp = consensusState.GetTimestamp() + timeoutTimestamp

	var msgs []sdk.Msg

	for j := 0; j < msgnum; j++ {
		msg := ibctypes.NewMsgTransfer(srcPort, srcChannel, coin, sender, receiver, timeoutHeight, timeoutTimestamp)
		msgs = append(msgs, msg)

	}
	return msgs
}

func grpcclient() {
	connV, err := grpc.Dial("competition.bharvest.io:9090", grpc.WithInsecure(), grpc.WithBlock())
	grpcConn = connV
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
}

func main() {

	var txnum int = 1  // 총 tx = txnum * 계정수
	var msgnum int = 1 //1tx 당 msg수 /
	var round int = 1  // 총실행횟수= txnum * round
	var srcPort string = ""
	var srcChannel string = ""
	coin, err := sdk.ParseCoinNormalized("1uatom")

	if grpcConn == nil {
		grpcclient()
	}

	defer grpcConn.Close()

	keyring, err := keys.New("swapchain", "os", "/home/ubuntu/.liquidityapp/", nil)
	if err != nil {
		log.Fatalf("did not keyring: %v", err)
	}

	keylist, _ := keyring.List()

	wait := new(sync.WaitGroup)
	wait.Add(len(keylist))

	for i, key := range keylist {
		sender := key.GetAddress()
		receiver := key.GetAddress()
		accountarmor, err := keyring.ExportPrivKeyArmor(key.GetName(), "qwer1234")
		if err != nil {
			println(err)
		}
		accountpriv, _, err := crypto.UnarmorDecryptPrivKey(accountarmor, "qwer1234")
		if err != nil {
			println(err)
		}
		if i%2 == 0 {
			go signtxsend(round, txnum, msgnum, accountpriv, wait, srcPort, srcChannel, coin, sender, receiver.String())
		} else {
			go signtxsend(round, txnum, msgnum, accountpriv, wait, srcPort, srcChannel, coin, sender, receiver.String())
		}
	}

	wait.Wait()

}
