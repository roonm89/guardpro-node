package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"
)

const TotalSupply = 100000000
const Ticker = "GuardPro"
const WalletFile = "guardpro.priv"

type Transaction struct {
	Emisor    string   `json:"emisor"`
	Receptor  string   `json:"receptor"`
	Monto     float64  `json:"monto"`
	Timestamp int64    `json:"timestamp"`
	FirmaR    *big.Int `json:"firma_r"`
	FirmaS    *big.Int `json:"firma_s"`
}

type Block struct {
	Index         int           `json:"index"`
	Timestamp     string        `json:"timestamp"`
	Transacciones []Transaction `json:"transacciones"`
	PrevHash      string        `json:"prev_hash"`
	Hash          string        `json:"hash"`
}

var Blockchain []Block
var Mempool []Transaction

func CalcularDobleHash(b Block) string {
	txBytes, _ := json.Marshal(b.Transacciones)
	record := string(rune(b.Index)) + b.Timestamp + string(txBytes) + b.PrevHash
	pHash := sha256.Sum256([]byte(record))
	dHash := sha256.Sum256(pHash[:])
	return hex.EncodeToString(dHash[:])
}

func ObtenerSaldo(direccion string) float64 {
	saldo := 0.0
	if direccion == "GP_CREADOR" {
		saldo = TotalSupply
	}
	
	for _, bloque := range Blockchain {
		for _, tx := range bloque.Transacciones {
			if tx.Emisor == direccion {
				saldo -= tx.Monto
			}
			if tx.Receptor == direccion {
				saldo += tx.Monto
			}
		}
	}
	return saldo
}

func CrearTransaccion(privKey *ecdsa.PrivateKey, emisor string, receptor string, monto float64) (Transaction, error) {
	saldoDisponible := ObtenerSaldo(emisor)
	if emisor != "GP_CREADOR" && saldoDisponible < monto {
		return Transaction{}, fmt.Errorf("fondos insuficientes. Saldo actual: %.2f %s", saldoDisponible, Ticker)
	}

	tx := Transaction{
		Emisor:    emisor,
		Receptor:  receptor,
		Monto:     monto,
		Timestamp: time.Now().Unix(),
	}

	txData := fmt.Sprintf("%s->%s:%.2f@%d", tx.Emisor, tx.Receptor, tx.Monto, tx.Timestamp)
	hash := sha256.Sum256([]byte(txData))
	
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return Transaction{}, err
	}
	tx.FirmaR = r
	tx.FirmaS = s

	return tx, nil
}

func CargarOGenerarBilletera() (*ecdsa.PrivateKey, string, error) {
	if _, err := os.Stat(WalletFile); err == nil {
		bytesPrivados, err := os.ReadFile(WalletFile)
		if err != nil {
			return nil, "", err
		}
		privKey, err := x509.ParseECPrivateKey(bytesPrivados)
		if err != nil {
			return nil, "", err
		}
		pubKeyHash := sha256.Sum256(elliptic.Marshal(elliptic.P256(), privKey.PublicKey.X, privKey.PublicKey.Y))
		direccionBilletera := "GP" + hex.EncodeToString(pubKeyHash[:16])
		return privKey, direccionBilletera, nil
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	bytesPrivados, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, "", err
	}
	_ = os.WriteFile(WalletFile, bytesPrivados, 0600)
	pubKeyHash := sha256.Sum256(elliptic.Marshal(elliptic.P256(), privKey.PublicKey.X, privKey.PublicKey.Y))
	direccionBilletera := "GP" + hex.EncodeToString(pubKeyHash[:16])
	return privKey, direccionBilletera, nil
}

func main() {
	fmt.Printf("--- [Fase 4] Inicializando Motor de Transacciones Guard Pro 7 (%s) ---\n\n", Ticker)

	privateKey, miDireccion, _ := CargarOGenerarBilletera()
	fmt.Printf("🔑 Tu Billetera Local: %s\n", miDireccion)

	// 1. Inicializar cadena con bloque Génesis vacío
	genesisBlock := Block{0, time.Now().String(), []Transaction{}, "", ""}
	genesisBlock.Hash = CalcularDobleHash(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)
	fmt.Println("📦 Bloque Génesis acoplado a la memoria.")

	// 2. Reclamar tus primeras monedas desde el fondo común
	fmt.Println("\n💸 [Simulación]: Reclamando tus primeras monedas desde el fondo común...")
	txInicial, _ := CrearTransaccion(privateKey, "GP_CREADOR", miDireccion, 5000.0)
	Mempool = append(Mempool, txInicial)
	fmt.Printf("✅ Transacción añadida a la Mempool: Creador -> Tu Dirección (5000.00 %s)\n", Ticker)

	// 3. Forzar el empaquetado del Bloque #1 para asentar tus 5,000 monedas en el libro contable
	fmt.Println("\n⚒️ [Procesamiento]: Cerrando el Bloque #1 para validar tus fondos...")
	bloque1 := Block{
		Index:         len(Blockchain),
		Timestamp:     time.Now().String(),
		Transacciones: Mempool,
		PrevHash:      Blockchain[len(Blockchain)-1].Hash,
	}
	bloque1.Hash = CalcularDobleHash(bloque1)
	Blockchain = append(Blockchain, bloque1)
	Mempool = []Transaction{} // Limpiar mempool
	fmt.Printf("🔒 Bloque #1 asegurado en la cadena. Saldo asentado: %.2f %s\n", ObtenerSaldo(miDireccion), Ticker)

	// 4. Ahora que tu saldo real es de 5,000, el motor te autorizará este envío perfectamente
	billeteraAmigo := "GP8f3e2b9a1c4d5e6f7g8h9i0j1k2l3m4n"
	fmt.Printf("\n📤 [Transacción]: Enviando monedas desde tu nodo hacia un nodo auxiliar (%s)...\n", billeteraAmigo)
	txEnvio, err := CrearTransaccion(privateKey, miDireccion, billeteraAmigo, 1250.50)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}
	Mempool = append(Mempool, txEnvio)
	fmt.Println("✅ Envío firmado criptográficamente y colocado en la Mempool.")

	// 5. Meter el envío de tu amigo en el Bloque #2
	fmt.Println("\n⚒️ [Procesamiento]: Empaquetando la transferencia en el Bloque #2...")
	bloque2 := Block{
		Index:         len(Blockchain),
		Timestamp:     time.Now().String(),
		Transacciones: Mempool,
		PrevHash:      Blockchain[len(Blockchain)-1].Hash,
	}
	bloque2.Hash = CalcularDobleHash(bloque2)
	Blockchain = append(Blockchain, bloque2)
	Mempool = []Transaction{}

	fmt.Printf("🔒 Bloque #2 creado de forma exitosa. Hash: %s\n", bloque2.Hash)

	// 6. Verificar los saldos contables finales de tu ecosistema financiero
	fmt.Println("\n📊 [Libro de Contabilidad General - Saldos Finales]")
	fmt.Printf("▪️ Saldo definitivo de tu Billetera: %.2f %s\n", ObtenerSaldo(miDireccion), Ticker)
	fmt.Printf("▪️ Saldo definitivo de tu Amigo: %.2f %s\n", ObtenerSaldo(billeteraAmigo), Ticker)
}
