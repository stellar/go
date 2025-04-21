package token_transfer

import "github.com/stellar/go/xdr"

func NewMemoFromXdrMemo(m *xdr.Memo) *MuxedInfo {
	protoMemo := &MuxedInfo{}

	switch m.Type {
	case xdr.MemoTypeMemoNone:
		return nil
	case xdr.MemoTypeMemoText:
		protoMemo.Content = &MuxedInfo_Text{
			Text: *m.Text,
		}
	case xdr.MemoTypeMemoId:
		protoMemo.Content = &MuxedInfo_Id{
			Id: uint64(*m.Id),
		}
	case xdr.MemoTypeMemoHash:
		hashSlice := make([]byte, 32)
		copy(hashSlice, (*m.Hash)[:])
		protoMemo.Content = &MuxedInfo_Hash{
			Hash: hashSlice,
		}
	case xdr.MemoTypeMemoReturn:
		hashSlice := make([]byte, 32)
		copy(hashSlice, (*m.RetHash)[:])
		protoMemo.Content = &MuxedInfo_Hash{
			Hash: hashSlice,
		}
	}
	return protoMemo
}

func NewMemoFromId(id uint64) *MuxedInfo {
	return &MuxedInfo{
		Content: &MuxedInfo_Id{
			Id: id,
		},
	}
}
