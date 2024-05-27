package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgWithdrawCreditRetry{}

func NewMsgWithdrawCreditRetry(creditorAddr string, poolID uint64, withdrawalID uint64) *MsgWithdrawCreditRetry {
	return &MsgWithdrawCreditRetry{
		PoolId:           poolID,
		WithdrawalId:     withdrawalID,
		CreditorAddress: creditorAddr,
	}
}

func (msg MsgWithdrawCreditRetry) Route() string {
	return RouterKey
}

func (msg MsgWithdrawCreditRetry) Type() string {
	return "WithdrawCreditRetry"
}

func (msg *MsgWithdrawCreditRetry) GetSigners() []sdk.AccAddress {
	creditor := sdk.MustAccAddressFromBech32(msg.GetCreditorAddress())
	return []sdk.AccAddress{creditor}
}

func (msg *MsgWithdrawCreditRetry) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgWithdrawCreditRetry) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress()); err != nil {
		return ErrInvalidCreditorAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetWithdrawalId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
