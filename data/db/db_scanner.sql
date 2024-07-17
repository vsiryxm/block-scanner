-- 区块表
DROP TABLE IF EXISTS `tb_block`;
CREATE TABLE `tb_block` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `block_number` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '区块号',
  `block_hash` varchar(66) NOT NULL DEFAULT '' COMMENT '区块哈希',
  `block_time` datetime NOT NULL COMMENT '区块时间',
  `scanned_at` datetime NOT NULL COMMENT '扫描时间',
  `scan_status` tinyint(4) unsigned NOT NULL DEFAULT 0 COMMENT '扫描状态: 0未完成, 1已完成',
  PRIMARY KEY (`id`),
  UNIQUE KEY `block_number` (`block_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 交易哈希表
DROP TABLE IF EXISTS `tb_transaction`;
CREATE TABLE `tb_transaction` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `block_number` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '区块号',
  `tx_hash` varchar(66) NOT NULL DEFAULT '' COMMENT '交易哈希',
  `from_address` varchar(42) NOT NULL DEFAULT '' COMMENT '发送方地址',
  `to_address` varchar(42) NOT NULL DEFAULT '' COMMENT '接收方地址',
  `contract_address` varchar(42) NOT NULL DEFAULT '' COMMENT '相关的合约地址',
  `tx_time` datetime NOT NULL COMMENT '交易时间',
  `gas_used` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '使用的gas量',
  `gas_price` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT 'gas价格',
  `status` tinyint(4) unsigned NOT NULL DEFAULT 0 COMMENT '交易状态: 1成功, 0失败',
  `logs` text DEFAULT NULL COMMENT '交易日志',
  `created_at` datetime NOT NULL DEFAULT current_timestamp() COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `tx_hash` (`tx_hash`),
  KEY `block_number` (`block_number`),
  KEY `contract_address` (`contract_address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;