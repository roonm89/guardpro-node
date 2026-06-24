package main

import (
	"bufio"
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
	"strconv"
	"strings"
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
	FirmaR        *big.Int      `json:"firma_r"`
	FirmaS        *big.Int      `json:"firma_s"`
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
	fmt.Printf("=== INTERFAZ INTERACTIVA GUARD PRO 7 (%s) ===\n", Ticker)
	privateKey, miDireccion, _ := CargarOGenerarBilletera()
	fmt.Printf("🔑 Billetera Local: %s\n", miDireccion)

	// Inicializar Génesis pasándole nil a las firmas del bloque
	genesisBlock := Block{0, time.Now().String(), []Transaction{}, "", "", nil, nil}
	genesisBlock.Hash = CalcularDobleHash(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	// Simular saldo inicial empaquetado pasándole nil a las firmas del bloque
	txInicial, _ := CrearTransaccion(privateKey, "GP_CREADOR", miDireccion, 5000.0)
	Mempool = append(Mempool, txInicial)
	bloque1 := Block{1, time.Now().String(), Mempool, genesisBlock.Hash, "", nil, nil}
	bloque1.Hash = CalcularDobleHash(bloque1)
	Blockchain = append(Blockchain, bloque1)
	Mempool = []Transaction{}

	fmt.Println("🚀 Nodo en línea de forma interactiva. Escribe 'ayuda' para ver comandos.")
	
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nguardpro> ")
		if !scanner.Scan() {
			break
		}
		entrada := strings.TrimSpace(strings.ToLower(scanner.Text()))
		partes := strings.Fields(entrada)

		if len(partes) == 0 {
			continue
		}

		switch partes[0] {
		case "ayuda":
			fmt.Println("📜 Comandos Disponibles:")
			fmt.Println("  saldo       - Muestra las monedas de tu billetera local")
			fmt.Println("  cadena      - Despliega todo el historial de bloques en JSON")
			fmt.Println("  enviar      - Realiza una transferencia. Uso: enviar [dirección] [monto]")
			fmt.Println("  mempool     - Muestra transacciones en sala de espera")
			fmt.Println("  minar       - Empaqueta la mempool actual en un nuevo bloque")
			fmt.Println("  salir       - Apaga el nodo de forma segura")
		case "saldo":
			fmt.Printf("💰 Saldo Actual: %.2f %s\n", ObtenerSaldo(miDireccion), Ticker)
		case "cadena":
			cadenaJSON, _ := json.MarshalIndent(Blockchain, "", "  ")
			fmt.Println(string(cadenaJSON))
		case "mempool":
			if len(Mempool) == 0 {
				fmt.Println("📭 La Mempool está vacía. No hay transacciones pendientes.")
			} else {
				mempoolJSON, _ := json.MarshalIndent(Mempool, "", "  ")
				fmt.Println(string(mempoolJSON))
			}
		case "enviar":
			if len(partes) < 3 {
				fmt.Println("❌ Uso incorrecto. Formato: enviar [dirección] [monto]")
				continue
			}
			receptor := partes[1]
			monto, err := strconv.ParseFloat(partes[2], 64)
			if err != nil || monto <= 0 {
				fmt.Println("❌ Monto inválido.")
				continue
			}
			tx, err := CrearTransaccion(privateKey, miDireccion, receptor, monto)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				Mempool = append(Mempool, tx)
				fmt.Printf("✅ Transacción firmada colocada en Mempool (Enviando %.2f a %s)\n", monto, receptor)
			}
		case "minar":
			if len(Mempool) == 0 {
				fmt.Println("❌ No hay transacciones en la Mempool para empaquetar.")
				continue
			}
			nuevoBloque := Block{
				Index:         len(Blockchain),
				Timestamp:     time.Now().String(),
				Transacciones: Mempool,
				PrevHash:      Blockchain[len(Blockchain)-1].Hash,
				Hash:          "",
				FirmaR:        nil,
				FirmaS:        nil,
			}
			nuevoBloque.Hash = CalcularDobleHash(nuevoBloque)
			Blockchain = append(Blockchain, nuevoBloque)
			Mempool = []Transaction{}
			fmt.Printf("🔒 Bloque #%d Creado Exitosamente. Hash: %s\n", nuevoBloque.Index, nuevoBloque.Hash)
		case "salir":
			fmt.Println("👋 Cerrando consola interactiva de GuardPro de forma segura. ¡Descansa!")
			return
		default:
			fmt.Println("❌ Comando no reconocido. Escribe 'ayuda' para ver la lista.")
		}
	}
}
