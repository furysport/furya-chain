package types

func NewGenesisState(params Params, nextPoolID uint64, nextCreditID uint64, nextPrizeID uint64, nexWithdrawalID uint64) *GenesisState {
	return &GenesisState{
		Params:           params,
		NextPoolId:       nextPoolID,
		NextCreditId:    nextCreditID,
		NextPrizeId:      nextPrizeID,
		NextWithdrawalId: nexWithdrawalID,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		NextPoolId:       1,
		NextCreditId:    1,
		NextPrizeId:      1,
		NextWithdrawalId: 1,
	}
}

func ValidateGenesis(data GenesisState) error {
	return nil
}
