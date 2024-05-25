package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

var _ sdk.Msg = &MsgWithdrawPayment{}

func NewMsgWithdrawPayment(paymentor string, toAddress string, poolID uint64, paymentID uint64) *MsgWithdrawPayment {
	return &MsgWithdrawPayment{
		PoolId:           poolID,
		PaymentId:        paymentID,
		PaymentorAddress: paymentor,
		ToAddress:        toAddress,
	}
}

func (msg MsgWithdrawPayment) Route() string {
	return RouterKey
}

func (msg MsgWithdrawPayment) Type() string {
	return "WithdrawPayment"
}

func (msg *MsgWithdrawPayment) GetSigners() []sdk.AccAddress {
	paymentor := sdk.MustAccAddressFromBech32(msg.GetPaymentorAddress())
	return []sdk.AccAddress{paymentor}
}

func (msg *MsgWithdrawPayment) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgWithdrawPayment) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress()); err != nil {
		return ErrInvalidPaymentorAddress
	}

	// Verify that a valid bech32 address is passed regardless of its prefix
	// because the toAddress can be targeting the local zone or the pool remote zone
	if _, _, err := bech32.DecodeAndConvert(msg.GetToAddress()); err != nil {
		return ErrInvalidDestinationAddress
	}

	if msg.GetPoolId() == UnknownID || msg.GetPaymentId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
