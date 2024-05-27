package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic validates if a credit has a valid poolID, creditID and creditorAddress
// meaning that it can be stored
func (credit *Credit) ValidateBasic() error {
	if credit.PoolId == UnknownID {
		return errorsmod.Wrapf(ErrInvalidID, "pool ID")
	}
	if credit.CreditId == UnknownID {
		return errorsmod.Wrapf(ErrInvalidID, "credit ID")
	}
	if credit.State == CreditState_Unspecified {
		return errorsmod.Wrapf(ErrInvalidCreditState, "no state specified")
	}
	if _, err := sdk.AccAddressFromBech32(credit.CreditorAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidCreditorAddress, err.Error())
	}
	if _, err := sdk.AccAddressFromBech32(credit.WinnerAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidWinnerAddress, err.Error())
	}
	return nil
}
