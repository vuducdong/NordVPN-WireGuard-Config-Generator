package validator

import (
	"strings"

	"nordgen/internal/types"
)

func IsHex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < 64; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func IsKey(s string) bool {
	if len(s) != 44 {
		return false
	}
	if s[43] != '=' {
		return false
	}
	for i := 0; i < 43; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '+' || c == '/') {
			return false
		}
	}
	return true
}

func IsIPv4(s string) bool {
	if len(s) == 0 {
		return false
	}
	dots := 0
	num := 0
	hasNum := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if !hasNum || num > 255 {
				return false
			}
			dots++
			num = 0
			hasNum = false
		} else if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
			if num > 255 {
				return false
			}
			hasNum = true
		} else {
			return false
		}
	}
	return dots == 3 && hasNum && num <= 255
}

func ParseCommon(key, dns, endpoint string, keepAlive *int) (types.ValidatedConfig, []string) {
	var errs []string
	if key != "" && !IsKey(key) {
		errs = append(errs, "Invalid Private Key")
	}

	cleanDns := "103.86.96.100"
	if dns != "" {
		valid := true
		start := 0
		for i := 0; i <= len(dns); i++ {
			if i == len(dns) || dns[i] == ',' {
				part := strings.TrimSpace(dns[start:i])
				if !IsIPv4(part) {
					valid = false
					break
				}
				start = i + 1
			}
		}
		if !valid {
			errs = append(errs, "Invalid DNS IP")
		} else {
			cleanDns = dns
		}
	}

	if endpoint != "" && endpoint != "hostname" && endpoint != "station" {
		errs = append(errs, "Invalid endpoint type")
	}

	ka := 25
	if keepAlive != nil {
		if *keepAlive < 15 || *keepAlive > 120 {
			errs = append(errs, "Invalid keepalive")
		} else {
			ka = *keepAlive
		}
	}

	return types.ValidatedConfig{
		PrivateKey: key,
		DNS:        cleanDns,
		UseStation: endpoint == "station",
		KeepAlive:  ka,
	}, errs
}

func ValidateConfig(b types.ConfigRequest) (types.ValidatedConfig, string) {
	cfg, errs := ParseCommon(b.PrivateKey, b.DNS, b.Endpoint, b.KeepAlive)

	if b.Country == "" {
		errs = append(errs, "Missing country")
	}
	if b.City == "" {
		errs = append(errs, "Missing city")
	}
	if b.Name == "" {
		errs = append(errs, "Missing name")
	}

	cfg.Name = b.Name
	if len(errs) > 0 {
		return types.ValidatedConfig{}, strings.Join(errs, ", ")
	}
	return cfg, ""
}

func ValidateBatch(b types.BatchConfigReq) (types.ValidatedConfig, string) {
	cfg, errs := ParseCommon(b.PrivateKey, b.DNS, b.Endpoint, b.KeepAlive)
	if len(errs) > 0 {
		return types.ValidatedConfig{}, strings.Join(errs, ", ")
	}
	return cfg, ""
}
