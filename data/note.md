遇到的坑：

1、运行`from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)`，提示错误：
`Error getting transaction sender: transaction type not supported`

改成这样即可：
`from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)`

参考：https://github.com/ethereum/go-ethereum/issues/22918