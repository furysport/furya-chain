package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgWithdrawPaymentRetry{}

func NewMsgWithdrawPaymentRetry(paymentorAddr string, poolID uint64, withdrawalID uint64) *MsgWithdrawPaymentRetry {
	return &MsgWithdrawPaymentRetry{
		PoolId:           poolID,
		WithdrawalId:     withdrawalID,
		PaymentorAddress: paymentorAddr,
	}
}

func (msg MsgWithdrawPaymentRetry) Route() string {
	return RouterKey
}

func (msg MsgWithdrawPaymentRetry) Type() string {
	return "WithdrawPaymentRetry"
}

func (msg *MsgWithdrawPaymentRetry) GetSigners() []sdk.AccAddress {
	paymentor := sdk.MustAccAddressFromBech32(msg.GetPaymentorAddress())
	return []sdk.AccAddress{paymentor}
}

func (msg *MsgWithdrawPaymentRetry) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgWithdrawPaymentRetry) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress()); err != nil {
		return ErrInvalidPaymentorAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetWithdrawalId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
