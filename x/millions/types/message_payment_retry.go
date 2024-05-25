package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgPaymentRetry{}

func NewMsgPaymentRetry(paymentorAddr string, poolID uint64, paymentID uint64) *MsgPaymentRetry {
	return &MsgPaymentRetry{
		PoolId:           poolID,
		PaymentId:        paymentID,
		PaymentorAddress: paymentorAddr,
	}
}

func (msg MsgPaymentRetry) Route() string {
	return RouterKey
}

func (msg MsgPaymentRetry) Type() string {
	return "PaymentRetry"
}

func (msg *MsgPaymentRetry) GetSigners() []sdk.AccAddress {
	paymentor := sdk.MustAccAddressFromBech32(msg.GetPaymentorAddress())
	return []sdk.AccAddress{paymentor}
}

func (msg *MsgPaymentRetry) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPaymentRetry) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress()); err != nil {
		return ErrInvalidPaymentorAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetPaymentId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
