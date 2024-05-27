package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCredit{}

func NewMsgCredit(creditor string, amount sdk.Coin, poolID uint64) *MsgCredit {
	return &MsgCredit{
		PoolId:           poolID,
		Amount:           amount,
		CreditorAddress: creditor,
	}
}

func (msg MsgCredit) Route() string {
	return RouterKey
}

func (msg MsgCredit) Type() string {
	return "Credit"
}

func (msg *MsgCredit) GetSigners() []sdk.AccAddress {
	creditor := sdk.MustAccAddressFromBech32(msg.GetCreditorAddress())
	return []sdk.AccAddress{creditor}
}

func (msg *MsgCredit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCredit) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress()); err != nil {
		return ErrInvalidCreditorAddress
	}

	// ensure denom is valid
	if err := sdk.ValidateDenom(msg.Amount.Denom); err != nil {
		return ErrInvalidCreditDenom
	}

	// If we have an amount, make sure it is not negative nor zero
	if msg.Amount.IsNil() || msg.Amount.Amount.LT(sdk.NewInt(MinAcceptableCreditAmount)) {
		return ErrInvalidCreditAmount
	}

	if msg.GetWinnerAddress() != "" {
		if _, err := sdk.AccAddressFromBech32(msg.GetWinnerAddress()); err != nil {
			return ErrInvalidWinnerAddress
		}
	}

	if msg.WinnerAddress != "" && msg.WinnerAddress != msg.CreditorAddress && msg.IsSponsor {
		return ErrInvalidSponsorWinnerCombo
	}

	if msg.GetPoolId() == UnknownID {
		return ErrInvalidID
	}

	return nil
}
