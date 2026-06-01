// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package localization

var EnglishMessages = map[string]string{
	"cli.debug.short":             "Debug a failed Soroban transaction",
	"cli.debug.long":              "Fetch and prepare a transaction for simulation",
	"cli.debug.example.basic":     "Glassbox debug <tx-hash>",
	"cli.debug.example.testnet":   "Glassbox debug --network testnet <tx-hash>",
	"cli.debug.example.gas_model": "Glassbox debug --gas-model ./custom-gas-model.json <tx-hash>",
	"cli.debug.flag.network":      "Stellar network to use (testnet, mainnet, futurenet)",
	"cli.debug.flag.rpc_url":      "Custom Horizon RPC URL",
	"cli.debug.flag.gas_model":    "Path to custom gas model JSON file",

	"cli.auth_debug.short":            "Debug multi-signature and threshold-based authorization failures",
	"cli.auth_debug.long":             "Analyze multi-signature authorization flows and identify failures",
	"cli.auth_debug.flag.detailed":    "Show detailed analysis and missing signatures",
	"cli.auth_debug.flag.json":        "Output as JSON",
	"cli.auth_debug.flag.custom_auth": "Path to custom auth configuration JSON",

	"error.invalid_network":       "invalid network: %s",
	"error.network_required":      "network must be one of: testnet, mainnet, futurenet",
	"error.fetch_transaction":     "failed to fetch transaction: %w",
	"error.parse_gas_model":       "failed to parse gas model: %w",
	"error.gas_model_validation":  "gas model validation failed: %s",
	"error.invalid_rpc_url":       "invalid RPC URL: %s",
	"error.transaction_not_found": "transaction not found: %s",

	"info.fetching_transaction":  "Fetching transaction for simulation",
	"info.gas_model_loaded":      "Gas model loaded and validated",
	"info.auth_analysis_started": "Fetching transaction for auth analysis",

	"output.transaction_envelope":    "Transaction Envelope: %d bytes",
	"output.custom_gas_model":        "Custom Gas Model Applied:",
	"output.network":                 "Network: %s",
	"output.total_costs":             "Total Costs: %d",
	"output.resource_limits":         "Resource Limits configured",
	"output.authorization_failed":    "[FAIL] Authorization FAILED",
	"output.authorization_succeeded": "[OK] Authorization SUCCEEDED",
	"output.summary_metrics":         "--- SUMMARY METRICS ---",
	"output.missing_signatures":      "--- MISSING SIGNATURES ---",
	"output.required_weight":         "required weight: %d",

	"validation.model_required":   "gas model file path cannot be empty",
	"validation.model_file_read":  "failed to read gas model file: %w",
	"validation.json_parse_error": "failed to parse gas model JSON: %w",

	"cli.keygen.short":               "Generate Ed25519 audit signing keys",
	"cli.keygen.long":                "Generate an Ed25519 key pair for audit log signing and export PEM files",
	"cli.keygen.flag.output_dir":     "Directory to write generated key files (default: current directory)",
	"cli.keygen.flag.key_name":       "Base filename for the generated key files (without extension)",
	"cli.keygen.flag.rotate":         "Generate a new key pair alongside any existing keys for rotation",
	"cli.keygen.flag.force":          "Overwrite existing key files without prompting",
	"info.keygen_generated":          "Key pair generated successfully",
	"info.keygen_private_key_file":   "Private key (PKCS#8 PEM): %s",
	"info.keygen_public_key_file":    "Public key  (SPKI PEM):   %s",
	"info.keygen_rotation_note":      "Rotation: keep the previous public key to verify existing signatures",
	"error.keygen_output_dir":        "failed to create output directory: %w",
	"error.keygen_write_private":     "failed to write private key file: %w",
	"error.keygen_write_public":      "failed to write public key file: %w",
	"error.keygen_file_exists":       "key file already exists (use --force to overwrite): %s",

	"cli.bench.short":              "Run performance benchmarks for RPC, replay, and source mapping",
	"cli.bench.long":               "Measure timing and memory for key pipeline stages",
	"cli.bench.flag.mode":          "Pipeline stage to benchmark: rpc, replay, sourcemap, or all",
	"cli.bench.flag.count":         "Number of benchmark iterations (default: 5)",
	"cli.bench.flag.json":          "Output benchmark results as JSON",
	"info.bench_running":           "Running %s benchmarks...",
	"info.bench_stage":             "Stage: %s",
	"info.bench_duration":          "  Duration (avg): %s",
	"info.bench_allocs":            "  Allocs/op:      %d",
	"info.bench_bytes":             "  Bytes/op:       %d",
	"error.bench_unknown_mode":     "unknown benchmark mode %q; choose rpc, replay, sourcemap, or all",

	"error.ledger_sequence_mismatch": "ledger sequence mismatch: transaction references %d but replay state is at %d",
	"info.ledger_recovery_attempt":   "Attempting recovery: fetching ledger snapshot for sequence %d",
	"info.ledger_recovery_success":   "Ledger snapshot retrieved for sequence %d; retrying replay",
	"error.ledger_recovery_failed":   "ledger sequence recovery failed: %w",
}

var SpanishMessages = map[string]string{
	"cli.debug.short":             "Depurar una transacción Soroban fallida",
	"cli.debug.long":              "Obtener y preparar una transacción para simulación",
	"cli.debug.example.basic":     "Glassbox debug <tx-hash>",
	"cli.debug.example.testnet":   "Glassbox debug --network testnet <tx-hash>",
	"cli.debug.example.gas_model": "Glassbox debug --gas-model ./modelo-gas-personalizado.json <tx-hash>",
	"cli.debug.flag.network":      "Red Stellar a utilizar (testnet, mainnet, futurenet)",
	"cli.debug.flag.rpc_url":      "URL de RPC personalizada",
	"cli.debug.flag.gas_model":    "Ruta al archivo JSON del modelo de gas personalizado",

	"cli.auth_debug.short":            "Depurar fallos de autorización con múltiples firmas",
	"cli.auth_debug.long":             "Analizar flujos de autorización con múltiples firmas",
	"cli.auth_debug.flag.detailed":    "Mostrar análisis detallado y firmas faltantes",
	"cli.auth_debug.flag.json":        "Salida como JSON",
	"cli.auth_debug.flag.custom_auth": "Ruta a archivo de configuración de autenticación",

	"error.invalid_network":       "red inválida: %s",
	"error.network_required":      "la red debe ser una de: testnet, mainnet, futurenet",
	"error.fetch_transaction":     "error al obtener transacción: %w",
	"error.parse_gas_model":       "error al analizar modelo de gas: %w",
	"error.gas_model_validation":  "validación de modelo de gas fallida: %s",
	"error.invalid_rpc_url":       "URL de RPC inválida: %s",
	"error.transaction_not_found": "transacción no encontrada: %s",

	"info.fetching_transaction":  "Obteniendo transacción para simulación",
	"info.gas_model_loaded":      "Modelo de gas cargado y validado",
	"info.auth_analysis_started": "Obteniendo transacción para análisis de autorización",

	"output.transaction_envelope":    "Envolvente de Transacción: %d bytes",
	"output.custom_gas_model":        "Modelo de Gas Personalizado Aplicado:",
	"output.network":                 "Red: %s",
	"output.total_costs":             "Costos Totales: %d",
	"output.resource_limits":         "Límites de Recursos configurados",
	"output.authorization_failed":    "[FALLO] Autorización FALLIDA",
	"output.authorization_succeeded": "[EXITO] Autorización EXITOSA",
	"output.summary_metrics":         "--- MÉTRICAS DE RESUMEN ---",
	"output.missing_signatures":      "--- FIRMAS FALTANTES ---",
	"output.required_weight":         "peso requerido: %d",

	"validation.model_required":   "la ruta del archivo de modelo de gas no puede estar vacía",
	"validation.model_file_read":  "error al leer archivo de modelo de gas: %w",
	"validation.json_parse_error": "error al analizar JSON del modelo de gas: %w",

	"cli.keygen.short":               "Generar claves Ed25519 para firma de auditoría",
	"cli.keygen.long":                "Genera un par de claves Ed25519 para firmar registros de auditoría",
	"cli.keygen.flag.output_dir":     "Directorio para escribir los archivos de claves",
	"cli.keygen.flag.key_name":       "Nombre base de los archivos de claves (sin extensión)",
	"cli.keygen.flag.rotate":         "Generar un nuevo par de claves junto a las existentes para rotación",
	"cli.keygen.flag.force":          "Sobrescribir archivos de claves existentes sin confirmación",
	"info.keygen_generated":          "Par de claves generado exitosamente",
	"info.keygen_private_key_file":   "Clave privada (PKCS#8 PEM): %s",
	"info.keygen_public_key_file":    "Clave pública  (SPKI PEM):   %s",
	"info.keygen_rotation_note":      "Rotación: conserve la clave pública anterior para verificar firmas existentes",
	"error.keygen_output_dir":        "error al crear directorio de salida: %w",
	"error.keygen_write_private":     "error al escribir archivo de clave privada: %w",
	"error.keygen_write_public":      "error al escribir archivo de clave pública: %w",
	"error.keygen_file_exists":       "el archivo de clave ya existe (use --force para sobrescribir): %s",

	"cli.bench.short":              "Ejecutar benchmarks de rendimiento para RPC, replay y source mapping",
	"cli.bench.long":               "Medir tiempo y memoria para etapas clave del pipeline",
	"cli.bench.flag.mode":          "Etapa del pipeline a medir: rpc, replay, sourcemap, o all",
	"cli.bench.flag.count":         "Número de iteraciones del benchmark",
	"cli.bench.flag.json":          "Mostrar resultados como JSON",
	"info.bench_running":           "Ejecutando benchmarks de %s...",
	"info.bench_stage":             "Etapa: %s",
	"info.bench_duration":          "  Duración (prom): %s",
	"info.bench_allocs":            "  Allocs/op:       %d",
	"info.bench_bytes":             "  Bytes/op:        %d",
	"error.bench_unknown_mode":     "modo de benchmark desconocido %q; elija rpc, replay, sourcemap, o all",

	"error.ledger_sequence_mismatch": "discrepancia de secuencia de ledger: la transacción referencia %d pero el estado de replay está en %d",
	"info.ledger_recovery_attempt":   "Intentando recuperación: obteniendo snapshot de ledger para la secuencia %d",
	"info.ledger_recovery_success":   "Snapshot de ledger obtenido para la secuencia %d; reintentando replay",
	"error.ledger_recovery_failed":   "recuperación de secuencia de ledger fallida: %w",
}

var ChineseMessages = map[string]string{
	"cli.debug.short":             "调试失败的 Soroban 交易",
	"cli.debug.long":              "获取并准备用于模拟的交易",
	"cli.debug.example.basic":     "Glassbox debug <tx-hash>",
	"cli.debug.example.testnet":   "Glassbox debug --network testnet <tx-hash>",
	"cli.debug.example.gas_model": "Glassbox debug --gas-model ./custom-gas-model.json <tx-hash>",
	"cli.debug.flag.network":      "使用的 Stellar 网络 (testnet, mainnet, futurenet)",
	"cli.debug.flag.rpc_url":      "自定义 Horizon RPC URL",
	"cli.debug.flag.gas_model":    "自定义 gas 模型 JSON 文件路径",

	"cli.auth_debug.short":            "调试多签名和阈值授权失败",
	"cli.auth_debug.long":             "分析多签名授权流程并识别失败",
	"cli.auth_debug.flag.detailed":    "显示详细分析和缺失签名",
	"cli.auth_debug.flag.json":        "输出为 JSON 格式",
	"cli.auth_debug.flag.custom_auth": "自定义身份验证配置 JSON 文件路径",

	"error.invalid_network":       "无效的网络: %s",
	"error.network_required":      "网络必须是以下之一: testnet, mainnet, futurenet",
	"error.fetch_transaction":     "获取交易失败: %w",
	"error.parse_gas_model":       "解析 gas 模型失败: %w",
	"error.gas_model_validation":  "gas 模型验证失败: %s",
	"error.invalid_rpc_url":       "无效的 RPC URL: %s",
	"error.transaction_not_found": "交易未找到: %s",

	"info.fetching_transaction":  "正在获取用于模拟的交易",
	"info.gas_model_loaded":      "Gas 模型已加载并验证",
	"info.auth_analysis_started": "正在获取用于授权分析的交易",

	"output.transaction_envelope":    "交易包: %d 字节",
	"output.custom_gas_model":        "自定义 Gas 模型已应用:",
	"output.network":                 "网络: %s",
	"output.total_costs":             "总成本: %d",
	"output.resource_limits":         "资源限制已配置",
	"output.authorization_failed":    "[失败] 授权失败",
	"output.authorization_succeeded": "[成功] 授权成功",
	"output.summary_metrics":         "--- 摘要指标 ---",
	"output.missing_signatures":      "--- 缺失签名 ---",
	"output.required_weight":         "所需权重: %d",

	"validation.model_required":   "gas 模型文件路径不能为空",
	"validation.model_file_read":  "读取 gas 模型文件失败: %w",
	"validation.json_parse_error": "解析 gas 模型 JSON 失败: %w",

	"cli.keygen.short":               "生成 Ed25519 审计签名密钥",
	"cli.keygen.long":                "生成用于审计日志签名的 Ed25519 密钥对并导出 PEM 文件",
	"cli.keygen.flag.output_dir":     "写入生成密钥文件的目录",
	"cli.keygen.flag.key_name":       "生成密钥文件的基础文件名（不含扩展名）",
	"cli.keygen.flag.rotate":         "在现有密钥旁生成新密钥对以进行轮换",
	"cli.keygen.flag.force":          "不提示直接覆盖现有密钥文件",
	"info.keygen_generated":          "密钥对生成成功",
	"info.keygen_private_key_file":   "私钥 (PKCS#8 PEM): %s",
	"info.keygen_public_key_file":    "公钥  (SPKI PEM):   %s",
	"info.keygen_rotation_note":      "轮换提示：保留旧公钥以验证现有签名",
	"error.keygen_output_dir":        "创建输出目录失败: %w",
	"error.keygen_write_private":     "写入私钥文件失败: %w",
	"error.keygen_write_public":      "写入公钥文件失败: %w",
	"error.keygen_file_exists":       "密钥文件已存在（使用 --force 覆盖）: %s",

	"cli.bench.short":              "运行 RPC、replay 和 source mapping 管道的性能基准测试",
	"cli.bench.long":               "测量关键管道阶段的时间和内存占用",
	"cli.bench.flag.mode":          "要测试的管道阶段: rpc, replay, sourcemap, 或 all",
	"cli.bench.flag.count":         "基准测试迭代次数",
	"cli.bench.flag.json":          "以 JSON 格式输出基准测试结果",
	"info.bench_running":           "正在运行 %s 基准测试...",
	"info.bench_stage":             "阶段: %s",
	"info.bench_duration":          "  平均耗时: %s",
	"info.bench_allocs":            "  每操作分配数: %d",
	"info.bench_bytes":             "  每操作字节数: %d",
	"error.bench_unknown_mode":     "未知基准测试模式 %q；请选择 rpc, replay, sourcemap, 或 all",

	"error.ledger_sequence_mismatch": "账本序列不匹配：交易引用 %d，但 replay 状态在 %d",
	"info.ledger_recovery_attempt":   "正在尝试恢复：获取序列 %d 的账本快照",
	"info.ledger_recovery_success":   "已获取序列 %d 的账本快照；正在重试 replay",
	"error.ledger_recovery_failed":   "账本序列恢复失败: %w",
}

func LoadTranslations() error {
	if err := RegisterMessages(English, EnglishMessages); err != nil {
		return err
	}
	if err := RegisterMessages(Spanish, SpanishMessages); err != nil {
		return err
	}
	if err := RegisterMessages(Chinese, ChineseMessages); err != nil {
		return err
	}
	return nil
}
