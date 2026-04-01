package main

import (
	"os"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// returns a CIDv1 for raw file bytes using a sha2-256 multihash.
func ComputeCID(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash, err := mh.SumStream(file, mh.SHA2_256, -1)
	if err != nil {
		return "", err
	}

	return cid.NewCidV1(cid.Raw, hash).String(), nil
}
