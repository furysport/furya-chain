package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

// RegisterLegacyAminoCodec Register the codec for the message passing
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Messages
	cdc.RegisterConcrete(&MsgDeposit{}, "furya/millions/MsgDeposit", nil)
	cdc.RegisterConcrete(&MsgDepositRetry{}, "furya/millions/MsgDepositRetry", nil)
	cdc.RegisterConcrete(&MsgDepositEdit{}, "furya/millions/MsgDepositEdit", nil)
	cdc.RegisterConcrete(&MsgClaimPrize{}, "furya/millions/MsgClaimPrize", nil)
	cdc.RegisterConcrete(&MsgDrawRetry{}, "furya/millions/MsgDrawRetry", nil)
	cdc.RegisterConcrete(&MsgWithdrawDeposit{}, "furya/millions/MsgWithdrawDeposit", nil)
	cdc.RegisterConcrete(&MsgWithdrawDepositRetry{}, "furya/millions/MsgWithdrawDepositRetry", nil)
	cdc.RegisterConcrete(&MsgRestoreInterchainAccounts{}, "furya/millions/MsgRestoreInterchainAccounts", nil)
	cdc.RegisterConcrete(&MsgGenerateSeed{}, "furya/millions/MsgGenerateSeed", nil)

	// Proposals
	cdc.RegisterConcrete(&ProposalUpdateParams{}, "furya/millions/ProposalUpdateParams", nil)
	cdc.RegisterConcrete(&ProposalRegisterPool{}, "furya/millions/ProposalRegisterPool", nil)
	cdc.RegisterConcrete(&ProposalUpdatePool{}, "furya/millions/ProposalUpdatePool", nil)
	cdc.RegisterConcrete(&ProposalClosePool{}, "furya/millions/ProposalClosePool", nil)
}

// RegisterInterfaces Register the implementations for the given codecs
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// Messages
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDeposit{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDepositRetry{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDepositEdit{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgClaimPrize{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgDrawRetry{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgWithdrawDeposit{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgWithdrawDepositRetry{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgRestoreInterchainAccounts{})
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgGenerateSeed{})

	// Proposals
	registry.RegisterImplementations((*govtypesv1beta1.Content)(nil), &ProposalUpdateParams{})
	registry.RegisterImplementations((*govtypesv1beta1.Content)(nil), &ProposalRegisterPool{})
	registry.RegisterImplementations((*govtypesv1beta1.Content)(nil), &ProposalUpdatePool{})
	registry.RegisterImplementations((*govtypesv1beta1.Content)(nil), &ProposalClosePool{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()

	RegisterLegacyAminoCodec(legacy.Cdc)
}
