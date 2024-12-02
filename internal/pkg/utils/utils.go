package utils

import (
	"math/big"
	"strconv"
	"strings"
)

func HexToUint64(hexStr string) uint64 {
	// 移除 "0x" 前缀
	hexStr = strings.TrimPrefix(hexStr, "0x")
	// 解析十六进制字符串
	result, _ := strconv.ParseUint(hexStr, 16, 64)
	return result
}

func HexToBigInt(hexStr string) *big.Int {
	bigInt := new(big.Int)
	bigInt.SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	return bigInt
}
