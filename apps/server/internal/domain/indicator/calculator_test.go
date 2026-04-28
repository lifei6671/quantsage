package indicator

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lifei6671/quantsage/apps/server/internal/domain/marketdata"
)

func TestCalculateDailyFactorsTableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bars    []marketdata.DailyBar
		assert  func(t *testing.T, items []DailyFactor)
		wantErr error
	}{
		{
			name: "ma window insufficient returns nil",
			bars: buildBars(
				[]string{"10", "11", "12", "13"},
				[]string{"100", "110", "120", "130"},
				nil,
			),
			assert: func(t *testing.T, items []DailyFactor) {
				t.Helper()
				if items[3].MA5 != nil {
					t.Fatalf("items[3].MA5 = %v, want nil", items[3].MA5)
				}
			},
		},
		{
			name: "ma5 and volume ratio",
			bars: buildBars(
				[]string{"10", "11", "12", "13", "14"},
				[]string{"100", "110", "120", "130", "140"},
				nil,
			),
			assert: func(t *testing.T, items []DailyFactor) {
				t.Helper()
				assertDecimalPtrString(t, items[4].MA5, "12")
				assertDecimalPtrString(t, items[4].VolumeMA5, "120")
				assertDecimalPtrString(t, items[4].VolumeRatio, "1.1667")
				assertBoolPtrValue(t, items[4].CloseAboveMA5, true)
			},
		},
		{
			name: "zero amplitude shadow ratio returns nil",
			bars: buildBars(
				[]string{"10", "11", "12", "13", "14"},
				[]string{"100", "110", "120", "130", "140"},
				func(i int, bar *marketdata.DailyBar) {
					if i == 4 {
						bar.Open = decimal.RequireFromString("14")
						bar.High = decimal.RequireFromString("14")
						bar.Low = decimal.RequireFromString("14")
						bar.Close = decimal.RequireFromString("14")
					}
				},
			),
			assert: func(t *testing.T, items []DailyFactor) {
				t.Helper()
				if items[4].UpperShadowRatio != nil {
					t.Fatalf("items[4].UpperShadowRatio = %v, want nil", items[4].UpperShadowRatio)
				}
				if items[4].LowerShadowRatio != nil {
					t.Fatalf("items[4].LowerShadowRatio = %v, want nil", items[4].LowerShadowRatio)
				}
			},
		},
		{
			name: "rsi returns 100 when window has gains without losses",
			bars: buildBars(
				[]string{"10", "11", "12", "13", "14", "15", "16"},
				[]string{"100", "110", "120", "130", "140", "150", "160"},
				nil,
			),
			assert: func(t *testing.T, items []DailyFactor) {
				t.Helper()
				assertDecimalPtrString(t, items[6].RSI6, "100")
			},
		},
		{
			name: "unsorted input returns error",
			bars: []marketdata.DailyBar{
				{
					TSCode:    "000001.SZ",
					TradeDate: time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
					Open:      decimal.RequireFromString("10"),
					High:      decimal.RequireFromString("11"),
					Low:       decimal.RequireFromString("9"),
					Close:     decimal.RequireFromString("10"),
					Vol:       decimal.RequireFromString("100"),
				},
				{
					TSCode:    "000001.SZ",
					TradeDate: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
					Open:      decimal.RequireFromString("10"),
					High:      decimal.RequireFromString("11"),
					Low:       decimal.RequireFromString("9"),
					Close:     decimal.RequireFromString("10"),
					Vol:       decimal.RequireFromString("100"),
				},
			},
			wantErr: errBarsNotSorted,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			items, err := CalculateDailyFactors(tc.bars)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("CalculateDailyFactors() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CalculateDailyFactors() error = %v", err)
			}

			tc.assert(t, items)
		})
	}
}

func buildBars(closes, volumes []string, mutator func(index int, bar *marketdata.DailyBar)) []marketdata.DailyBar {
	items := make([]marketdata.DailyBar, 0, len(closes))
	for i := range closes {
		closePrice := decimal.RequireFromString(closes[i])
		bar := marketdata.DailyBar{
			TSCode:    "000001.SZ",
			TradeDate: time.Date(2026, 4, 1+i, 0, 0, 0, 0, time.UTC),
			Open:      closePrice.Sub(decimal.RequireFromString("0.2")),
			High:      closePrice.Add(decimal.RequireFromString("0.5")),
			Low:       closePrice.Sub(decimal.RequireFromString("0.5")),
			Close:     closePrice,
			PreClose:  closePrice.Sub(decimal.RequireFromString("0.1")),
			Change:    decimal.RequireFromString("0.1"),
			PctChg:    decimal.RequireFromString("1"),
			Vol:       decimal.RequireFromString(volumes[i]),
			Amount:    decimal.RequireFromString("1000000"),
			Source:    "sample",
		}
		if mutator != nil {
			mutator(i, &bar)
		}
		items = append(items, bar)
	}

	return items
}

func assertDecimalPtrString(t *testing.T, value *decimal.Decimal, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("value = nil, want %s", want)
	}
	if value.String() != want {
		t.Fatalf("value.String() = %s, want %s", value.String(), want)
	}
}

func assertBoolPtrValue(t *testing.T, value *bool, want bool) {
	t.Helper()
	if value == nil {
		t.Fatalf("value = nil, want %t", want)
	}
	if *value != want {
		t.Fatalf("*value = %t, want %t", *value, want)
	}
}
