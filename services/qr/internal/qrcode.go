package internal

import (
	"net/http"

	"github.com/aaronarduino/goqrsvg"
	svg "github.com/ajstarks/svgo"
	"github.com/boombuler/barcode/qr"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
)

type qrCodeHandler struct {
	Logger *supportlog.Entry
}

type qrCodeRequest struct {
	Address *keypair.FromAddress `path:"address"`
}

func (h qrCodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := qrCodeRequest{}
	err := httpdecode.DecodePath(r, &req)
	if err != nil || req.Address == nil {
		h.Logger.Errorf("Error: %#v", err)
		badRequest.Render(w)
		return
	}

	qrCode, _ := qr.Encode(req.Address.Address(), qr.M, qr.AlphaNumeric)

	qs := goqrsvg.NewQrSVG(qrCode, 5)

	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)
	qs.StartQrSVG(s)
	qs.WriteQrSVG(s)
	s.End()
}
