module github.com/b-harvest/swapbot

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.42.4
	github.com/tendermint/liquidity v1.2.4
	google.golang.org/grpc v1.35.0
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
