package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Decimal int64

func (d Decimal) Float64() float64 {
	x := float64(d)
	x = x / 100
	return x
}

func (d Decimal) String() string {
	x := float64(d)
	x = x / 100
	return fmt.Sprintf("%.2f", x)
}

func (d Decimal) SetString(s string) Decimal {
	x, _ := strconv.ParseFloat(s, 64)
	x = x * 100
	return Decimal(int64(x))
}

func NewDecial(s string) Decimal {
	x, _ := strconv.ParseFloat(s, 64)
	x = x * 100
	return Decimal(int64(x))
}

func NewDecialFromFloat64(f float64) Decimal {
	x := f * 100
	return Decimal(int64(x))
}

type OfxTransaction struct {
	FitID          string    `json:`
	Type           string    `json:`
	PostedDateTime time.Time `json:`
	UserDateTime   time.Time `json:`
	Amount         Decimal   `json:`
	Memo           string    `json:`
}

func (t OfxTransaction) String() string {
	return fmt.Sprintf("FitID:%-15s Type:%-10s User:%s Amount: $%8s Memo:%s\n",
		t.FitID, t.Type, t.PostedDateTime.Format("2006/01/02"), t.Amount, t.Memo,
	)
}

type Ofx struct {
	GeneratedDateTime        time.Time         `json:`
	Language                 string            `json:`
	AccountBankNumber        string            `json:`
	AccountNumber            string            `json:`
	AccountType              string            `json:`
	Currency                 string            `json:`
	LedgerBalance            Decimal           `json:`
	AvailiableBalance        Decimal           `json:`
	TransactionStartDateTime time.Time         `json:`
	TrnasactionEndDateTime   time.Time         `json:`
	Transactions             []*OfxTransaction `json:`
}

func (o Ofx) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Generated:%s Lang:%s AccountBankNumber:%s AccountNumber:%s AccountType:%s\n",
		o.GeneratedDateTime, o.Language, o.AccountBankNumber, o.AccountNumber, o.AccountType))
	buf.WriteString(fmt.Sprintf("Ledger: $%s Av: $%s Start:%s End%s\n",
		o.LedgerBalance, o.AvailiableBalance, o.TransactionStartDateTime, o.TrnasactionEndDateTime))

	for _, t := range o.Transactions {
		buf.WriteString(fmt.Sprintf("%s", t))
	}

	return buf.String()
}

type nextKey int

const (
	none            nextKey = iota
	acctID          nextKey = iota
	acctType        nextKey = iota
	curDef          nextKey = iota
	branchID        nextKey = iota
	bankID          nextKey = iota
	transAmount     nextKey = iota
	transDatePosted nextKey = iota
	transUserDate   nextKey = iota
	transFitID      nextKey = iota
	transDesc       nextKey = iota
	transMemo       nextKey = iota
	transType       nextKey = iota
	legerBal        nextKey = iota
	AvailBal        nextKey = iota
)

func Parse(f io.Reader) (*Ofx, error) {
	ofx := &Ofx{Transactions: []*OfxTransaction{}}
	stack := make([]string, 1000)
	stackPos := 0

	next := none
	var trans *OfxTransaction = nil

	dec := xml.NewDecoder(f)

	tok, err := dec.RawToken()
	for err == nil {
		switch t := tok.(type) {
		case xml.StartElement:
			stack[stackPos] = t.Name.Local
			stackPos++

			switch t.Name.Local {
			case "ACCTID":
				next = acctID

			case "BRANCHID":
				next = branchID

			case "BANKID":
				next = bankID

			case "ACCTTYPE":
				next = acctType

			case "CURDEF":
				next = curDef

			case "STMTTRN":
				trans = &OfxTransaction{}

			case "DTPOSTED":
				next = transDatePosted

			case "FITID":
				next = transFitID

			case "TRNAMT":
				next = transAmount

			case "NAME":
				next = transDesc
			case "MEMO":
				next = transMemo

			case "TRNTYPE":
				next = transType

			case "LEDGERBAL":
				next = legerBal

			case "AVAILBAL":
				next = AvailBal
			}

		case xml.CharData:
			var b bytes.Buffer
			if _, err := b.Write(t); err != nil {
				return nil, err
			}
			res := strings.TrimSpace(b.String())

			switch next {
			case acctID:
				ofx.AccountNumber = res

			// case branchID:
			//	ofx.BranchCode = res

			case bankID:
				ofx.AccountBankNumber = res

			case transDesc:
				trans.Memo = res

			case transMemo:
				trans.Memo = res

			case transFitID:
				trans.FitID = res

			case curDef:
				ofx.Currency = res

			case acctType:
				ofx.AccountType = res

			case transDatePosted:
				if len(res) < 8 {
					return nil, fmt.Errorf("Invalid date posted string: '%s'", res)
				}
				res = res[:8]
				// YYYYMMDD
				if t, err := time.Parse("20060102", res); err != nil {
					return nil, err
				} else {
					trans.PostedDateTime = t
				}

			case transAmount:
				trans.Amount = NewDecial(res)

			case transType:
				trans.Type = res

			case legerBal:
				ofx.LedgerBalance = NewDecial(res)
			case AvailBal:
				ofx.AvailiableBalance = NewDecial(res)
			}

			next = none

		case xml.EndElement:
			for stackPos != 0 {
				if stack[stackPos-1] == "STMTTRN" {
					ofx.Transactions = append(ofx.Transactions, trans)
					trans = nil
				}

				if stack[stackPos-1] == t.Name.Local {
					stackPos--
					break
				}
				stackPos--
			}

		default:
			log.Printf("Unknown: %T %s\n", t, t)
		}

		tok, err = dec.RawToken()

		if err != nil && err != io.EOF {
			log.Printf("Error: %s\n", err)
		}
	}

	return ofx, nil

}

func main() {

	o, err := Parse(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to parse input, error: %v\n", err)
		os.Exit(1)
	}

	res, err := json.Marshal(o)

	if err != nil {
		log.Fatalf("Failed to Marshal into json, error: %v\n", err)
		os.Exit(2)
	}

	fmt.Println(string(res))

}
