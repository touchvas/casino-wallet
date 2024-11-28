package wallet

const DecimalMultiplierNone = 1
const DecimalMultiplierTen = 10
const DecimalMultiplierHundreds = 100
const DecimalMultiplierThousands = 1000
const DecimalMultiplierTenOfThousands = 10000

type DecimalMultiplier int64

func (e DecimalMultiplier) In64() int64 {
	switch e {

	case DecimalMultiplierNone:
		return 1
	case DecimalMultiplierTen:
		return DecimalMultiplierTen
	case DecimalMultiplierHundreds:
		return DecimalMultiplierHundreds
	case DecimalMultiplierThousands:
		return DecimalMultiplierThousands
	case DecimalMultiplierTenOfThousands:
		return DecimalMultiplierTenOfThousands
	default:
		return DecimalMultiplierNone
	}
}
