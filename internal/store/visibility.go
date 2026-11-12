package store

import "strings"

func MissingRequiredExchangeCredentialFields(exchangeType, apiKey, secretKey, passphrase, hyperliquidWalletAddr, asterUser, asterSigner, asterPrivateKey, lighterWalletAddr, lighterAPIKeyPrivateKey string) []string {
	switch strings.ToLower(strings.TrimSpace(exchangeType)) {
	case "binance", "bybit", "gate", "indodax":
		return missingNamedFields(
			namedField{"api_key", apiKey},
			namedField{"secret_key", secretKey},
		)
	case "okx", "bitget", "kucoin":
		return missingNamedFields(
			namedField{"api_key", apiKey},
			namedField{"secret_key", secretKey},
			namedField{"passphrase", passphrase},
		)
	case "hyperliquid":
		return missingNamedFields(
			namedField{"api_key", apiKey},
			namedField{"hyperliquid_wallet_addr", hyperliquidWalletAddr},
		)
	case "aster":
		return missingNamedFields(
			namedField{"aster_user", asterUser},
			namedField{"aster_signer", asterSigner},
			namedField{"aster_private_key", asterPrivateKey},
		)
	case "lighter":
		return missingNamedFields(
			namedField{"lighter_wallet_addr", lighterWalletAddr},
			namedField{"lighter_api_key_private_key", lighterAPIKeyPrivateKey},
		)
	default:
		return []string{"exchange_type"}
	}
}

type namedField struct {
	name  string
	value string
}

func missingNamedFields(fields ...namedField) []string {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}
