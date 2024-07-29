package utils

import (
	"fmt"
	"math/big"
)

func FormatEther(wei *big.Int) string {
	// 1 Ether = 10^18 Wei
	exp := big.NewInt(18)
	ether := new(big.Float).SetInt(wei)

	// 创建 10^18 的大数
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), exp, nil))

	// 进行除法运算
	ether.Quo(ether, divisor)

	// 格式化为字符串，保留 18 位小数
	return fmt.Sprintf("%.18f", ether)
}

func FormatEtherToInt64(wei *big.Int) int64 {
	// 1 Ether = 10^18 Wei
	exp := big.NewInt(18)
	divisor := new(big.Int).Exp(big.NewInt(10), exp, nil)

	// 进行除法运算
	ether := new(big.Int).Div(wei, divisor)

	// 转换为 int64
	return ether.Int64()
}

func FormatEtherToFloat64(wei *big.Int) float64 {
	// 1 Ether = 10^18 Wei
	exp := big.NewInt(18)
	ethValue := new(big.Float).SetInt(wei)

	// 创建 10^18 的大数
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), exp, nil))

	// 进行除法运算
	ethValue.Quo(ethValue, divisor)

	// 转换为 float64
	result, _ := ethValue.Float64()

	return result
}
