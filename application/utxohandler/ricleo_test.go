package utxohandler

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/MadBase/MadNet/application/objs"
	"github.com/MadBase/MadNet/application/objs/uint256"
	"github.com/MadBase/MadNet/consensus/db"
	cobjs "github.com/MadBase/MadNet/consensus/objs"
	"github.com/MadBase/MadNet/constants"
	"github.com/MadBase/MadNet/crypto"
	"github.com/dgraph-io/badger/v2"
)

func makeTransfer(t *testing.T, sender objs.Signer, receiver objs.Signer, transferAmount uint64, v *objs.ValueStore) *objs.Tx {
	txIn, err := v.MakeTxIn()
	if err != nil {
		t.Fatal(err)
	}
	value, err := v.Value()
	vuint64, err := value.ToUint64()
	returnedAmount := vuint64 - transferAmount
	value = &uint256.Uint256{}
	_, _ = value.FromUint64(returnedAmount)
	value2 := &uint256.Uint256{}
	_, _ = value2.FromUint64(transferAmount)

	if err != nil {
		t.Fatal(err)
	}
	chainID, err := txIn.ChainID()
	if err != nil {
		t.Fatal(err)
	}
	receiverPubkey, err := receiver.Pubkey()
	if err != nil {
		t.Fatal(err)
	}

	senderPubkey, err := sender.Pubkey()
	if err != nil {
		t.Fatal(err)
	}

	tx := &objs.Tx{}
	tx.Vin = []*objs.TXIn{txIn}
	newValueStoreSender := &objs.ValueStore{
		VSPreImage: &objs.VSPreImage{
			ChainID:  chainID,
			Value:    value,
			Owner:    &objs.ValueStoreOwner{SVA: objs.ValueStoreSVA, CurveSpec: constants.CurveSecp256k1, Account: crypto.GetAccount(senderPubkey)},
			TXOutIdx: 0,
		},
		TxHash: make([]byte, 32),
	}

	// the new utxo that will be generated by this transaction
	newValueStoreReceiver := &objs.ValueStore{
		VSPreImage: &objs.VSPreImage{
			ChainID:  chainID,
			Value:    value2,
			Owner:    &objs.ValueStoreOwner{SVA: objs.ValueStoreSVA, CurveSpec: constants.CurveSecp256k1, Account: crypto.GetAccount(receiverPubkey)},
			TXOutIdx: 1,
		},
		TxHash: make([]byte, 32),
	}
	newUTXOSender := &objs.TXOut{}
	err = newUTXOSender.NewValueStore(newValueStoreSender)
	if err != nil {
		t.Fatal(err)
	}

	newUTXOReceiver := &objs.TXOut{}
	err = newUTXOReceiver.NewValueStore(newValueStoreReceiver)
	if err != nil {
		t.Fatal(err)
	}
	tx.Vout = append(tx.Vout, newUTXOSender, newUTXOReceiver)
	err = tx.SetTxHash() // <- compute the root from the TxHash smt
	if err != nil {
		t.Fatal(err)
	}
	err = v.Sign(tx.Vin[0], sender)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func GenerateBlock(chain []*cobjs.BClaims, stateRoot []byte, txHshLst [][]byte) ([]*cobjs.BClaims, error) {
	var prevBlock []byte
	var headerRoot []byte
	if len(chain) == 0 {
		chain = []*cobjs.BClaims{}
		prevBlock = crypto.Hasher([]byte("foo"))
		headerRoot = crypto.Hasher([]byte(""))
	} else {
		_prevBlock, err := chain[len(chain)-1].BlockHash()
		if err != nil {
			return nil, err
		}
		prevBlock = _prevBlock
		headerRoot = crypto.Hasher([]byte("")) // todo: how to generate the block smt
	}
	txRoot, err := cobjs.MakeTxRoot(txHshLst) // generating the smt root
	log.Printf("txRoot height: (%d): %x\n", len(chain)+1, txRoot)
	if err != nil {
		if err != nil {
			return nil, err
		}
	}
	bclaims := &cobjs.BClaims{
		ChainID:    1,
		Height:     uint32(len(chain) + 1),
		TxCount:    uint32(len(txHshLst)),
		PrevBlock:  prevBlock,
		TxRoot:     txRoot,
		StateRoot:  stateRoot,
		HeaderRoot: headerRoot,
	}
	chain = append(chain, bclaims)

	log.Printf(
		"\nBlock: {\n\tChainID: %d\n\tHeight: %d\n\tTxCount: %d\n\tPrevBlock: %x\n\tTxRoot: %x\n\tStateRoot: %x\n\tHeaderRoot: %x\n}\n\n",
		bclaims.ChainID,
		bclaims.Height,
		bclaims.TxCount,
		bclaims.PrevBlock,
		bclaims.TxRoot,
		bclaims.StateRoot,
		bclaims.HeaderRoot,
	)
	return chain, nil
}

func getAllStateMerkleProofs(hndlr *UTXOHandler, txs []*objs.Tx) func(txn *badger.Txn) error {
	fn := func(txn *badger.Txn) error {
		stateTrie, err := hndlr.GetTrie().GetCurrentTrie(txn)
		if err != nil {
			return err
		}
		log.Printf("Trie height: %d\n", stateTrie.TrieHeight)
		for _, tx := range txs {
			txHash, err := tx.TxHash()
			if err != nil {
				return err
			}
			log.Println("===========Proof of inclusion=========")
			log.Printf("Tx: %x\n", txHash)
			log.Println("======================================")
			utxoIDs, err := tx.GeneratedUTXOID()
			if err != nil {
				return err
			}
			for i, utxoID := range utxoIDs {
				//auditPath, included, proofKey, proofVal, err := stateTrie.MerkleProof(txn, utxoID) // *badger.Txn, key []byte
				bitmap, auditPath, proofHeight, included, proofKey, proofVal, err := stateTrie.MerkleProofCompressed(txn, utxoID)
				if err != nil {
					return err
				}
				mproof := &db.MerkleProof{
					Included:  included,
					KeyHeight: proofHeight,
					Key:       proofKey,
					Value:     proofVal,
					Bitmap:    bitmap,
					Path:      auditPath,
				}
				mpbytes, err := mproof.MarshalBinary()
				if err != nil {
					return err
				}
				log.Printf("UTXOID: %x\n", utxoID)
				log.Printf("auditPath: %x\n", auditPath)
				log.Printf("Bitmap: %x\n", bitmap)
				log.Printf("Proof height: %x\n", proofHeight)
				log.Print("Included:", included)
				log.Printf("Proof key: %x\n", proofKey)
				log.Printf("Proof value: %x\n", proofVal)
				log.Printf("Proof capnproto: %x\n", mpbytes)
				if len(utxoIDs) > i+1 {
					log.Println("---------------------")
				}
			}
			log.Println("======================================")
			log.Println()

		}
		return nil
	}
	return fn
}

func TestRicLeo(t *testing.T) {
	// Database setup
	log.Println("TestRicLeo starting")
	dir, err := ioutil.TempDir("", "badger-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()
	opts := badger.DefaultOptions(dir)
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////////////////////////////////////////
	signer := &crypto.Secp256k1Signer{}
	err = signer.SetPrivk(crypto.Hasher([]byte("secret")))

	if err != nil {
		t.Fatal(err)
	}

	signer2 := &crypto.Secp256k1Signer{}
	err = signer2.SetPrivk(crypto.Hasher([]byte("secret2")))

	if err != nil {
		t.Fatal(err)
	}

	hndlr := NewUTXOHandler(db)
	err = hndlr.Init(1)
	if err != nil {
		t.Fatal(err)
	}

	///////// Block 1 ////////////
	log.Println("Block 1:")
	// Creating First UTXO
	var txs []*objs.Tx
	var deposits []*objs.ValueStore
	var txHshLst [][]byte
	for i := uint64(0); i < 5; i++ {
		value := &uint256.Uint256{}
		_, _ = value.FromUint64(i + 1)
		deposits = append(deposits, makeDeposit(t, signer, 1, int(i), value)) // created pre-image object
		txs = append(txs, makeTxs(t, signer, deposits[i]))
		txHash, err := txs[i].TxHash()
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("Tx hash (%d): %x", i, txHash)
		txHshLst = append(txHshLst, txHash)
	}

	var stateRoot []byte
	err = db.Update(func(txn *badger.Txn) error {
		stateRoot, err = hndlr.ApplyState(txn, txs, 1)
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("stateRoot: %x\n", stateRoot)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(getAllStateMerkleProofs(hndlr, txs))
	if err != nil {
		t.Fatal(err)
	}

	// Generating block 1
	chain, err := GenerateBlock(nil, stateRoot, txHshLst)
	if err != nil {
		t.Fatal(err)
	}

	////////// Block 2 /////////////
	// this is consuming utxo generated on block 1
	log.Println("Block 2:")
	tx2 := makeTransfer(t, signer, signer2, 1, deposits[1])
	txHash2, err := tx2.TxHash()
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Tx hash: %x", txHash2)
	err = db.Update(func(txn *badger.Txn) error {
		stateRoot, err = hndlr.ApplyState(txn, []*objs.Tx{tx2}, 2)
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("stateRoot2: %x\n", stateRoot)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(getAllStateMerkleProofs(hndlr, []*objs.Tx{tx2}))
	if err != nil {
		t.Fatal(err)
	}

	chain, err = GenerateBlock(chain, stateRoot, [][]byte{txHash2})
	if err != nil {
		t.Fatal(err)
	}

	//////////// Generating cobjs.BClaims and PClaims ////////////////////
	bnVal := &crypto.BNGroupValidator{}
	if err != nil {
		t.Fatal(err)
	}
	bclaims := chain[0]
	bhsh, err := bclaims.BlockHash()
	if err != nil {
		t.Fatal(err)
	}
	gk := &crypto.BNGroupSigner{}
	gk.SetPrivk(crypto.Hasher([]byte("secret")))
	sig, err := gk.Sign(bhsh)
	if err != nil {
		t.Fatal(err)
	}
	bh := &cobjs.BlockHeader{
		BClaims:  bclaims,
		SigGroup: sig,
		TxHshLst: txHshLst,
	}
	err = bh.ValidateSignatures(bnVal)
	if err != nil {
		t.Fatal(err)
	}
	rcert, err := bh.GetRCert()
	if err != nil {
		t.Fatal(err)
	}
	err = rcert.ValidateSignature(bnVal)
	if err != nil {
		t.Fatal(err)
	}
	bclaimsBin, err := chain[0].MarshalBinary()
	log.Printf("BClaim block 1: %x", bclaimsBin)
	log.Printf("SigGrup Block 1: %x", rcert.SigGroup)

	pclms := &cobjs.PClaims{
		BClaims: chain[1],
		RCert:   rcert,
	}

	pClaimsBin, err := pclms.MarshalBinary()
	log.Printf("PClaims Block 2: %x", pClaimsBin)
	prop := &cobjs.Proposal{
		PClaims:  pclms,
		TxHshLst: [][]byte{txHash2},
	}
	err = prop.Sign(signer)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Sig PClaims Block 2: %x", prop.Signature)
}