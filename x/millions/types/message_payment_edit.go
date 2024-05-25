package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgPaymentEdit{}

func NewMsgPaymentEdit(paymentorAddr string, poolID uint64, paymentID uint64) *MsgPaymentEdit {
	return &MsgPaymentEdit{
		PoolId:           poolID,
		PaymentId:        paymentID,
		PaymentorAddress: paymentorAddr,
	}
}

func (msg MsgPaymentEdit) Route() string {
	return RouterKey
}

func (msg MsgPaymentEdit) Type() string {
	return "PaymentEdit"
}

func (msg *MsgPaymentEdit) GetSigners() []sdk.AccAddress {
	paymentor := sdk.MustAccAddressFromBech32(msg.GetPaymentorAddress())
	return []sdk.AccAddress{paymentor}
}

func (msg *MsgPaymentEdit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPaymentEdit) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress()); err != nil {
		return ErrInvalidPaymentorAddress
	}

	if msg.GetWinnerAddress() != "" {
		if _, err := sdk.AccAddressFromBech32(msg.GetWinnerAddress()); err != nil {
			return ErrInvalidWinnerAddress
		}
	}

	if msg.GetPoolId() == UnknownID || msg.GetPaymentId() == UnknownID {
		return ErrInvalidID
	}

	if msg.IsSponsor != nil {
		if msg.WinnerAddress != "" && msg.WinnerAddress != msg.PaymentorAddress && msg.IsSponsor.Value {
			return ErrInvalidSponsorWinnerCombo
		}
	}

	return nil
}
