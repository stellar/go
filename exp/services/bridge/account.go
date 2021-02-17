package main

type Account struct {
	MemoID   int64  `pg:"memo_id,pk"`
	Username string `pg:"username"`
}
