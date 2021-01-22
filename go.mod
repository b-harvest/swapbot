module github.com/b-harvest/swapbot

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/tendermint/liquidity v0.1.0-rc0.0.20210113093033-1631ad4a86cc
	google.golang.org/grpc v1.33.2
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
