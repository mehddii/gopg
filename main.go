package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("PGDSN")
	db, err := sql.Open("pgx", dsn)

	if err != nil || db.Ping() != nil {
		log.Fatal("Could not connect to the database.", err)
	}
	defer db.Close()

	log.Println("Successfully connected to the database.")

	var aid, bid, abalance, filler string
	err = db.QueryRow(`SELECT * FROM pgbench_accounts LIMIT 1`).Scan(
		&aid,
		&bid,
		&abalance,
		&filler,
	)

	if err != nil {
		panic(err)
	}

	log.Printf(
		"'%v' | '%v' | '%v' | '%v'",
		aid,
		bid,
		abalance,
		filler,
	)

	// find the data file
	var relPath string
	err = db.QueryRow(`SELECT pg_relation_filepath('pgbench_accounts')`).Scan(&relPath)

	if err != nil {
		panic(err)
	}

	relPath = "/var/lib/postgres/data/" + relPath
	log.Print(relPath)

	file, err := os.OpenFile(relPath, os.O_RDONLY, os.ModeTemporary)
	if err != nil {
		panic(err)
	}

	page0Header := make([]byte, 24)
	n, err := file.Read(page0Header)

	if err != nil {
		panic(err)
	}

	log.Println(n, "bytes where read (page 0's header).")
	log.Println(page0Header)

	var p PageHeaderData
	err = p.ParseHeader(page0Header)
	if err != nil {
		panic(err)
	}

	log.Println(p)

	sz := uint(p.Lower - 24)
	page0ItemId := make([]byte, sz)

	n, err = file.Read(page0ItemId)
	if err != nil {
		panic(err)
	}
	log.Println(n, "bytes where read.")
	// log.Println(page0ItemId)

	i := NewItemId(sz)
	err = i.ParseItemId(page0ItemId)
	if err != nil {
		panic(err)
	}

	log.Println(i)
}

type PageHeaderData struct {
	Lsn             uint64 // needs special decoding (*)
	Checksum        int16
	Flags           uint16
	Lower           uint16
	Upper           uint16
	Special         uint16
	PagesizeVersion uint16 // *
	Xid             uint32
}

func (p *PageHeaderData) ParseHeader(header []byte) error {
	if len(header) != 24 {
		return fmt.Errorf("Expected 24 bytes header.")
	}

	return binary.Read(bytes.NewReader(header), binary.LittleEndian, p)
}

type ItemIdData struct {
	LpOff   uint16
	LPFlags uint16
	LpLen   uint16
}

type ItemId []uint32

func NewItemId(sz uint) ItemId {
	return make([]uint32, sz/4)
}

func (i *ItemId) ParseItemId(lps []byte) error {
	return binary.Read(bytes.NewReader(lps), binary.LittleEndian, i)
}
