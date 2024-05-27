package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCreditEdit{}

func NewMsgCreditEdit(creditorAddr string, poolID uint64, creditID uint64) *MsgCreditEdit {
	return &MsgCreditEdit{
		PoolId:           poolID,
		CreditId:        creditID,
		CreditorAddress: creditorAddr,
	}
}

func (msg MsgCreditEdit) Route() string {
	return RouterKey
}

func (msg MsgCreditEdit) Type() string {
	return "CreditEdit"
}

func (msg *MsgCreditEdit) GetSigners() []sdk.AccAddress {
	creditor := sdk.MustAccAddressFromBech32(msg.GetCreditorAddress())
	return []sdk.AccAddress{creditor}
}

func (msg *MsgCreditEdit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreditEdit) ValidateBasic() error {
	// Ensure the address is correct and that we are able to acquire it
	if _, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress()); err != nil {
		return ErrInvalidCreditorAddress
	}

	if msg.GetWinnerAddress() != "" {
		if _, err := sdk.AccAddressFromBech32(msg.GetWinnerAddress()); err != nil {
			return ErrInvalidWinnerAddress
		}
	}

	if msg.GetPoolId() == UnknownID || msg.GetCreditId() == UnknownID {
		return ErrInvalidID
	}

	if msg.IsSponsor != nil {
		if msg.WinnerAddress != "" && msg.WinnerAddress != msg.CreditorAddress && msg.IsSponsor.Value {
			return ErrInvalidSponsorWinnerCombo
		}
	}

	return nil
}
