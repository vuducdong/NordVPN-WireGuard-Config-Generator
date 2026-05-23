package qr

import (
	go_qr "github.com/piglig/go-qr"
)

func GenerateSVG(data []byte) ([]byte, error) {
	code, err := go_qr.EncodeBinary(data, go_qr.Medium)
	if err != nil {
		return nil, err
	}
	cfg := go_qr.NewQrCodeImgConfig(4, 4, go_qr.WithOptimalSVG())
	return code.ToSVGBytes(cfg)
}
