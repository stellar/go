package token_transfer

import "github.com/stellar/go/xdr"

func NewMemoFromXdrMemo(m *xdr.Memo) *Memo {
	protoMemo := &Memo{}

	switch m.Type {
	case xdr.MemoTypeMemoText:
		if m.Text != nil {
			protoMemo.Content = &Memo_Text{
				Text: *m.Text,
			}
		}
	case xdr.MemoTypeMemoId:
		if m.Id != nil {
			protoMemo.Content = &Memo_Id{
				Id: uint64(*m.Id),
			}
		}
	case xdr.MemoTypeMemoHash:
		if m.Hash != nil {
			hashSlice := make([]byte, 32)
			copy(hashSlice, (*m.Hash)[:])
			protoMemo.Content = &Memo_Hash{
				Hash: hashSlice,
			}
		}
	case xdr.MemoTypeMemoReturn:
		if m.RetHash != nil {
			hashSlice := make([]byte, 32)
			copy(hashSlice, (*m.RetHash)[:])
			protoMemo.Content = &Memo_Hash{
				Hash: hashSlice,
			}
		}
	}

	return protoMemo
}

func NewMemoFromId(id uint64) *Memo {
	return &Memo{
		Content: &Memo_Id{
			Id: id,
		},
	}
}
