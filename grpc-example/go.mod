module github.com/b-harvest/swapbot

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.40.0
	github.com/gogo/protobuf v1.3.2 // indirect
	google.golang.org/grpc v1.33.2
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
