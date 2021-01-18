package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	//"github.com/cosmos/cosmos-sdk/simapp"
	//"github.com/cosmos/cosmos-sdk/testutil/testdata"
	//cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	//"github.com/cosmos/cosmos-sdk/types/tx/signing"
	//xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

//https://github.com/cosmos/cosmos-sdk/blob/master/docs/run-node/txs.md
//https://github.com/p2p-org/relayerGun/blob/gun/relayer/chain.go

func queryState() error {
	myAddress, err := sdk.AccAddressFromBech32("cosmos1z36q8ddla8zmjyaxmdwpzlj3srwe45d8pzc2ug")
	if err != nil {
		return err
	}

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",    // your gRPC server address.
		grpc.WithInsecure(), // The SDK doesn't support any transport security mechanism.
	)
	defer grpcConn.Close()

	// This creates a gRPC client to query the x/bank service.
	bankClient := banktypes.NewQueryClient(grpcConn)
	bankRes, err := bankClient.Balance(
		context.Background(),
		&banktypes.QueryBalanceRequest{Address: myAddress.String(), Denom: "uatom"},
	)
	if err != nil {
		return err
	}

	fmt.Println(bankRes.GetBalance()) // Prints the account balance

	// This creates a gRPC client to query the x/bank service.
	authClient := authtypes.NewQueryClient(grpcConn)
	authRes, err := authClient.Account(
		context.Background(),
		&authtypes.QueryAccountRequest{Address: myAddress.String()},
	)
	if err != nil {
		return err
	}

	fmt.Println(authRes.GetAccount()) // Prints the account balance

	return nil
}

/*
func banksend() error {

    priv1, _, addr1 := testdata.KeyTestPubAddr()
    priv2, _, addr2 := testdata.KeyTestPubAddr()
    priv3, _, addr3 := testdata.KeyTestPubAddr()
    encCfg := simapp.MakeTestEncodingConfig()
    txBuilder := encCfg.TxConfig.NewTxBuilder()
    msg1 := banktypes.NewMsgSend(addr1, addr3, types.NewCoins(types.NewInt64Coin("uatom", 12)))
    msg2 := banktypes.NewMsgSend(addr2, addr3, types.NewCoins(types.NewInt64Coin("uatom", 34)))
    err := txBuilder.SetMsgs(msg1, msg2)
    if err != nil {
        return err
    }

    txBuilder.SetGasLimit(150000)
    txBuilder.SetFeeAmount(0.25)
    acc, err := auth.NewAccountRetriever(src.Cdc, src).GetAccount(src.MustGetAddress())
	if err != nil {
		return nil, err
	}
    privs := []cryptotypes.PrivKey{priv1, priv2}
    accNums:= []uint64{1,1} // The accounts' account numbers
    accSeqs:= []uint64{1,1} // The accounts' sequence numbers

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
    err := txBuilder.SetSignatures(sigsV2...)
    if err != nil {
        return err
    }

    // Second round: all signer infos are set, so each signer can sign.
    sigsV2 = []signing.SignatureV2{}
    for i, priv := range privs {
        signerData := xauthsigning.SignerData{
            ChainID:       chainID,
            AccountNumber: accNums[i],
            Sequence:      accSeqs[i],
        }
        sigV2, err := tx.SignWithPrivKey(
            encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData,
            txBuilder, priv, encCfg.TxConfig, accSeqs[i])
        if err != nil {
            return nil, err
        }

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

    grpcConn := grpc.Dial(
        "127.0.0.1:9090", // Or your gRPC server address.
        grpc.WithInsecure(), // The SDK doesn't support any transport security mechanism.
    )
    defer grpcConn.Close()

    // Broadcast the tx via gRPC. We create a new client for the Protobuf Tx
    // service.
    txClient := tx.NewServiceClient(grpcConn)
    // We then call the BroadcastTx method on this client.
    grpcRes, err := txClient.BroadcastTx(
        ctx,
        &tx.BroadcastTxRequest{
            Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
            TxBytes: txBytes, // Proto-binary of the signed transaction, see previous step.
        },
    )
    if err != nil {
        return err
    }

    fmt.Println(grpcRes.TxResponse.Code) // Should be `0` if the tx is successful

    return nil

}*/
func main() {
	queryState()
}
