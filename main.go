package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	swaptypes "github.com/tendermint/liquidity/x/liquidity/types"
	"google.golang.org/grpc"
)

var grpcConn *grpc.ClientConn

func signtxsend(round int, txnum int, msgnum int, priv cryptotypes.PrivKey, address sdk.AccAddress, w *sync.WaitGroup, tokenA string, tokenB string, swapamount int64) {
	defer w.Done()
	for i := 0; i < round; i++ {
		startTime := time.Now()
		var txBytes [][]byte

		accSeq, accNum := accountinfo(address)

		msgs := msgcreationbot(msgnum, address, tokenA, tokenB, swapamount)
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

func msgcreationbot(msgnum int, address sdk.AccAddress, tokenA string, tokenB string, swapamount int64) []sdk.Msg {

	var msgs []sdk.Msg
	var orderpirce sdk.Dec

	swapcoin := types.NewInt64Coin(tokenA, swapamount)
	orderpirce = orderPirce(tokenA, tokenB)

	for j := 0; j < msgnum; j++ {
		var orderpirceX sdk.Dec
		randtodec := sdk.NewDec(int64(rand.Intn(2)))
		pricepercentvalue := orderpirce.Mul(randtodec.Quo(sdk.NewDec(100)))
		if j%2 == 0 {
			orderpirceX = orderpirce.Add(pricepercentvalue)
		} else {
			orderpirceX = orderpirce.Sub(pricepercentvalue)
		}
		msg := swaptypes.NewMsgSwapWithinBatch(address, uint64(1), uint32(1), swapcoin, tokenB, orderpirceX, sdk.NewDecWithPrec(3, 3))
		msgs = append(msgs, msg)
		//println(orderpirceX.String())
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

func orderPirce(tokenA string, tokenB string) sdk.Dec {
	if grpcConn == nil {
		grpcclient()
	}
	var pool swaptypes.Pool
	liquClient := swaptypes.NewQueryClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	PoolRes, err := liquClient.LiquidityPool(
		ctx,
		&swaptypes.QueryLiquidityPoolRequest{PoolId: 1},
	)
	if err != nil {
		println(err)
	}
	pool = PoolRes.GetPool()

	reserveCoins := sdk.NewCoins()
	bankClient := banktypes.NewQueryClient(grpcConn)
	for _, denom := range pool.ReserveCoinDenoms {
		res, err := bankClient.Balance(ctx, banktypes.NewQueryBalanceRequest(sdk.AccAddress(pool.ReserveAccountAddress), denom))
		if err != nil && res.Balance.IsValid() {
			reserveCoins = reserveCoins.Add(*res.Balance)
		} else {
			fmt.Println(err)
		}
	}

	swapPrice := reserveCoins.AmountOf(tokenA).ToDec().Quo(reserveCoins.AmountOf(tokenB).ToDec())

	println("swapPrice:", swapPrice.String())
	return swapPrice
}

func main() {

	var txnum int = 100 // 총 tx = txnum * 계정수
	var msgnum int = 1  //1tx 당 msg수 /
	var round int = 2   // 총실행횟수= txnum * round
	var swapamount int64 = 1000
	var tokenA string = "uatom"
	var tokenB string = "uiris"

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

		accountarmor, err := keyring.ExportPrivKeyArmor(key.GetName(), "qwer1234")
		if err != nil {
			println(err)
		}
		accountpriv, _, err := crypto.UnarmorDecryptPrivKey(accountarmor, "qwer1234")
		if err != nil {
			println(err)
		}
		if i%2 == 0 {
			go signtxsend(round, txnum, msgnum, accountpriv, key.GetAddress(), wait, tokenA, tokenB, swapamount)
		} else {
			go signtxsend(round, txnum, msgnum, accountpriv, key.GetAddress(), wait, tokenB, tokenA, swapamount)
		}
	}

	wait.Wait()

}
