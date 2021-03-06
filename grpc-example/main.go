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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
)

//https://github.com/cosmos/cosmos-sdk/blob/master/docs/run-node/txs.md
//https://github.com/p2p-org/relayerGun/blob/gun/relayer/chain.go
//https://github.com/cosmos/cosmos-sdk/issues/8045
func queryState() error {
	myAddress, err := sdk.AccAddressFromBech32("cosmos1wlfjwg3ff8fy7qhut3eaj4agm8qpnw5ug7qjen")
	if err != nil {
		return err
	}

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090", // your gRPC server address.
		grpc.WithInsecure(),
		grpc.WithBlock(), // The SDK doesn't support any transport security mechanism.
	)
	//defer grpcConn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// This creates a gRPC client to query the x/bank service.
	bankClient := banktypes.NewQueryClient(grpcConn)
	bankRes, err := bankClient.AllBalances(
		ctx,
		&banktypes.QueryAllBalancesRequest{Address: myAddress.String()},
	)
	if err != nil {
		return err
	}

	fmt.Println(bankRes.GetBalances()) // Prints the account balance

	// This creates a gRPC client to query the x/bank service.
	authClient := authtypes.NewQueryClient(grpcConn)
	authRes, err := authClient.Account(
		ctx,
		&authtypes.QueryAccountRequest{Address: myAddress.String()},
	)
	if err != nil {
		return err
	}
	var acc authtypes.BaseAccount
	err = acc.Unmarshal(authRes.Account.Value)
	if err != nil {
		return err
	}
	fmt.Println(acc.Address)

	return nil
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
		//grpc.WaitForReady(false), grpc.FailFast(false),
	)
}

func banksend() error {
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",    // your gRPC server address.
		grpc.WithInsecure(), // The SDK doesn't support any transport security mechanism.
	)

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

	defer grpcConn.Close()
	//defer cancel()
	keyring, err := keys.New("swapchain", "os", "/root/.liquidityd/", nil)
	keylist, _ := keyring.List()
	for _, key := range keylist {
		println(key.GetAddress().String())
	}

	account1, err := keyring.Key("user1") //paswd
	account2, err := keyring.Key("validator")

	account1armor, err := keyring.ExportPrivKeyArmor("user1", "qwer1234")
	account2armor, err := keyring.ExportPrivKeyArmor("validator", "qwer1234")
	account1priv, _, err := crypto.UnarmorDecryptPrivKey(account1armor, "qwer1234")
	account2priv, _, err := crypto.UnarmorDecryptPrivKey(account2armor, "qwer1234")

	priv1, addr1 := account1priv, account1.GetAddress()
	priv2, addr2 := account2priv, account2.GetAddress()

	println(addr1.String())
	println(addr2.String())

	encCfg := simapp.MakeTestEncodingConfig()
	txBuilder := encCfg.TxConfig.NewTxBuilder()
	msg1 := banktypes.NewMsgSend(addr1, addr2, types.NewCoins(types.NewInt64Coin("uatom", 1000000)))
	msg2 := banktypes.NewMsgSend(addr2, addr1, types.NewCoins(types.NewInt64Coin("uusdt", 1000000)))
	var msgs []sdk.Msg
	msgs = append(msgs, msg1)
	msgs = append(msgs, msg2)
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return err
	}

	txBuilder.SetGasLimit(150000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uatom", sdk.NewInt(150))))

	if err != nil {
		return err
	}

	privs := []cryptotypes.PrivKey{priv1, priv2}
	authClient := authtypes.NewQueryClient(grpcConn)
	authRes1, err := authClient.Account(
		context.Background(),
		&authtypes.QueryAccountRequest{Address: addr1.String()},
	)
	if err != nil {
		return err
	}
	authRes2, err := authClient.Account(
		context.Background(),
		&authtypes.QueryAccountRequest{Address: addr2.String()},
	)
	if err != nil {
		return err
	}
	var acc1 authtypes.BaseAccount
	var acc2 authtypes.BaseAccount
	err = acc1.Unmarshal(authRes1.Account.Value)
	if err != nil {
		return err
	}
	err = acc2.Unmarshal(authRes2.Account.Value)
	if err != nil {
		return err
	}
	accNums := []uint64{acc1.AccountNumber, acc2.AccountNumber} // The accounts' account numbers
	accSeqs := []uint64{acc1.Sequence, acc2.Sequence}           // The accounts' sequence numbers
	startTime := time.Now()
	var txBytess [][]byte
	for q := 0; q < 10; q++ {
		// Broadcast the tx via gRPC. We create a new client for the Protobuf Tx
		// service.

		// First round: we gather all the signer infos. We use the "set empty
		// signature" hack to do that.
		var sigsV2 []signing.SignatureV2
		for i, priv := range privs {
			sigV2 := signing.SignatureV2{
				PubKey: priv.PubKey(),
				Data: &signing.SingleSignatureData{
					SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
					Signature: nil,
				},
				Sequence: accSeqs[i],
			}
			sigsV2 = append(sigsV2, sigV2)
		}

		err = txBuilder.SetSignatures(sigsV2...)
		if err != nil {
			return err
		}

		// Second round: all signer infos are set, so each signer can sign.
		sigsV2 = []signing.SignatureV2{}
		for i, priv := range privs {
			signerData := xauthsigning.SignerData{
				ChainID:       "swap-testnet-2001",
				AccountNumber: accNums[i],
				Sequence:      accSeqs[i],
			}
			sigV2, err := clienttx.SignWithPrivKey(
				encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData,
				txBuilder, priv, encCfg.TxConfig, accSeqs[i])
			if err != nil {
				return err
			}
			accSeqs[i] = accSeqs[i] + 1
			sigsV2 = append(sigsV2, sigV2)
		}
		err = txBuilder.SetSignatures(sigsV2...)
		if err != nil {
			return err
		}
		txBytes, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return err
		}
		txBytess = append(txBytess, txBytes)
		//sendtx(grpcConn, txBytes)

	}
	var count int = 1
	for _, txByte := range txBytess {
		sendtx(grpcConn, txByte)
		println(count)
		count = count + 1
	}
	elapsedTime := time.Since(startTime)

	fmt.Printf("실행시간: %s\n", elapsedTime)
	return nil

}
func main() {
	//queryState()
	banksend()
}
