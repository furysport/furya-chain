package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgPayment{}

func NewMsgPayment(paymentor string, amount sdk.Coin, poolID uint64) *MsgPayment {
	return &MsgPayment{
		PoolId:           poolID,
		Amount:           amount,
		PaymentorAddress: paymentor,
	}
}

func (msg MsgPayment) Route() string {
	return RouterKey
}

func (msg MsgPayment) Type() string {
	return "Payment"
}

func (msg *MsgPayment) GetSigners() []sdk.AccAddress {
	paymentor := sdk.MustAccAddressFromBech32(msg.GetPaymentorAddress())
	return []sdk.AccAddress{paymentor}
}

func (msg *MsgPayment) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgPayment) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress()); err != nil {
		return ErrInvalidPaymentorAddress
	}

	// ensure denom is valid
	if err := sdk.ValidateDenom(msg.Amount.Denom); err != nil {
		return ErrInvalidPaymentDenom
	}

	// If we have an amount, make sure it is not negative nor zero
	if msg.Amount.IsNil() || msg.Amount.Amount.LT(sdk.NewInt(MinAcceptablePaymentAmount)) {
		return ErrInvalidPaymentAmount
	}

	if msg.GetWinnerAddress() != "" {
		if _, err := sdk.AccAddressFromBech32(msg.GetWinnerAddress()); err != nil {
			return ErrInvalidWinnerAddress
		}
	}

	if msg.WinnerAddress != "" && msg.WinnerAddress != msg.PaymentorAddress && msg.IsSponsor {
		return ErrInvalidSponsorWinnerCombo
	}

	if msg.GetPoolId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
