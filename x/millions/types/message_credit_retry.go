package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCreditRetry{}

func NewMsgCreditRetry(creditorAddr string, poolID uint64, creditID uint64) *MsgCreditRetry {
	return &MsgCreditRetry{
		PoolId:           poolID,
		CreditId:        creditID,
		CreditorAddress: creditorAddr,
	}
}

func (msg MsgCreditRetry) Route() string {
	return RouterKey
}

func (msg MsgCreditRetry) Type() string {
	return "CreditRetry"
}

func (msg *MsgCreditRetry) GetSigners() []sdk.AccAddress {
	creditor := sdk.MustAccAddressFromBech32(msg.GetCreditorAddress())
	return []sdk.AccAddress{creditor}
}

func (msg *MsgCreditRetry) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreditRetry) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress()); err != nil {
		return ErrInvalidCreditorAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetCreditId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
