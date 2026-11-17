package rpc

import (
	"context"
	"testing"

	"github.com/owpengram/owpengram-td/clock"
	"github.com/owpengram/owpengram-td/tg"
	"go.uber.org/zap/zaptest"
)

func TestFemboyGramStarsDoNotExposeFiatTopupsOrGiftPurchases(t *testing.T) {
	if got := len(devStarsTopupOptions()); got != 0 {
		t.Fatalf("top-up options = %d, want none", got)
	}
	if got := len(devStarsGiftOptions()); got != 0 {
		t.Fatalf("Stars gift options = %d, want none", got)
	}
	if got := len(devStarsGiveawayOptions()); got != 0 {
		t.Fatalf("giveaway options = %d, want none", got)
	}

	r := New(Config{}, Deps{}, zaptest.NewLogger(t), clock.System)
	result, err := r.onPaymentsGetStarGifts(context.Background(), 0)
	if err != nil {
		t.Fatalf("get public star gifts: %v", err)
	}
	catalog, ok := result.(*tg.PaymentsStarGifts)
	if !ok {
		t.Fatalf("catalog = %T, want *tg.PaymentsStarGifts", result)
	}
	if len(catalog.Gifts) != 0 {
		t.Fatalf("public gift catalog contains %d purchasable gifts", len(catalog.Gifts))
	}
}

func TestPaidStarGiftInvoiceTypesAreRecognized(t *testing.T) {
	for _, invoice := range []tg.InputInvoiceClass{
		&tg.InputInvoiceStarGift{},
		&tg.InputInvoiceStarGiftUpgrade{},
		&tg.InputInvoiceStarGiftTransfer{},
		&tg.InputInvoiceStarGiftResale{},
		&tg.InputInvoiceStarGiftAuctionBid{},
		&tg.InputInvoiceStarGiftPrepaidUpgrade{},
		&tg.InputInvoiceStarGiftDropOriginalDetails{},
	} {
		if !isPaidStarGiftInvoice(invoice) {
			t.Errorf("%T was not recognized as a paid gift invoice", invoice)
		}
	}
	for _, invoice := range []tg.InputInvoiceClass{
		&tg.InputInvoiceStars{},
		&tg.InputInvoicePremiumGiftStars{},
	} {
		if isPaidStarGiftInvoice(invoice) {
			t.Errorf("%T was incorrectly classified as a paid star-gift invoice", invoice)
		}
	}
}
