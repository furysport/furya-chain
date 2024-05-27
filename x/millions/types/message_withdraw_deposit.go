package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

var _ sdk.Msg = &MsgWithdrawCredit{}

func NewMsgWithdrawCredit(creditor string, toAddress string, poolID uint64, creditID uint64) *MsgWithdrawCredit {
	return &MsgWithdrawCredit{
		PoolId:           poolID,
		CreditId:        creditID,
		CreditorAddress: creditor,
		ToAddress:        toAddress,
	}
}

func (msg MsgWithdrawCredit) Route() string {
	return RouterKey
}

func (msg MsgWithdrawCredit) Type() string {
	return "WithdrawCredit"
}

func (msg *MsgWithdrawCredit) GetSigners() []sdk.AccAddress {
	creditor := sdk.MustAccAddressFromBech32(msg.GetCreditorAddress())
	return []sdk.AccAddress{creditor}
}

func (msg *MsgWithdrawCredit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgWithdrawCredit) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress()); err != nil {
		return ErrInvalidCreditorAddress
	}

	// Verify that a valid bech32 address is passed regardless of its prefix
	// because the toAddress can be targeting the local zone or the pool remote zone
	if _, _, err := bech32.DecodeAndConvert(msg.GetToAddress()); err != nil {
		return ErrInvalidDestinationAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetCreditId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
