package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic validates if a payment has a valid poolID, paymentID and paymentorAddress
// meaning that it can be stored
func (payment *Payment) ValidateBasic() error {
	if payment.PoolId == UnknownID {
		return errorsmod.Wrapf(ErrInvalidID, "pool ID")
	}
	if payment.PaymentId == UnknownID {
		return errorsmod.Wrapf(ErrInvalidID, "payment ID")
	}
	if payment.State == PaymentState_Unspecified {
		return errorsmod.Wrapf(ErrInvalidPaymentState, "no state specified")
	}
	if _, err := sdk.AccAddressFromBech32(payment.PaymentorAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidPaymentorAddress, err.Error())
	}
	if _, err := sdk.AccAddressFromBech32(payment.WinnerAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidWinnerAddress, err.Error())
	}
	return nil
}
