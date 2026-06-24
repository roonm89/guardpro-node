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
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const TotalSupply = 100000000
const Ticker = "GuardPro"
const WalletFile = "guardpro.priv"
const PuertoRed = ":8080"

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

func ServidorRed() {
	escuchador, err := net.Listen("tcp", PuertoRed)
	if err != nil {
		return
	}
	defer escuchador.Close()

	for {
		conexion, err := escuchador.Accept()
		if err != nil {
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			json.NewEncoder(conn).Encode(Blockchain)
		}(conexion)
	}
}

// RelojUptime emite recompensas automaticas en segundo plano cada 10 segundos
func RelojUptime(privKey *ecdsa.PrivateKey, destino string) {
	cronometro := time.NewTicker(10 * time.Second)
	for range cronometro.C {
		// Crear transaccion de emision primaria (recompensa por estar activo)
		txRecompensa, _ := CrearTransaccion(privKey, "GP_CREADOR", destino, 10.0)
		
		// Forzar el empaquetado del bloque inmediatamente
		MempoolTransitoria := []Transaction{txRecompensa}
		nuevoBloque := Block{
			Index:         len(Blockchain),
			Timestamp:     time.Now().String(),
			Transacciones: MempoolTransitoria,
			PrevHash:      Blockchain[len(Blockchain)-1].Hash,
			Hash:          "",
			FirmaR:        nil,
			FirmaS:        nil,
		}
		nuevoBloque.Hash = CalcularDobleHash(nuevoBloque)
		Blockchain = append(Blockchain, nuevoBloque)
		
		// Imprimir aviso discreto interrumpiendo elegantemente la consola
		fmt.Printf("\n🪙 [RECOMPENSA]: +10.00 %s acreditados por Uptime. Bloque #%d minado.\nguardpro> ", Ticker, nuevoBloque.Index)
	}
}

func main() {
	fmt.Printf("=== SISTEMA DE EMISIÓN INTERACTIVA (%s) ===\n", Ticker)
	privateKey, miDireccion, _ := CargarOGenerarBilletera()
	fmt.Printf("🔑 Billetera Local: %s\n", miDireccion)

	// Inicializar Génesis
	genesisBlock := Block{0, time.Now().String(), []Transaction{}, "", "", nil, nil}
	genesisBlock.Hash = CalcularDobleHash(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	// Asignar fondo inicial estático de 5000 monedas
	txInicial, _ := CrearTransaccion(privateKey, "GP_CREADOR", miDireccion, 5000.0)
	Mempool = append(Mempool, txInicial)
	bloque1 := Block{1, time.Now().String(), Mempool, genesisBlock.Hash, "", nil, nil}
	bloque1.Hash = CalcularDobleHash(bloque1)
	Blockchain = append(Blockchain, bloque1)
	Mempool = []Transaction{}

	// Lanzar canales concurrentes en paralelo
	go ServidorRed()
	go RelojUptime(privateKey, miDireccion) // 🚀 NUEVO: Hilo de tiempo activo
	
	fmt.Printf("🌐 Servidor P2P activo. Reloj de Uptime configurado (Ciclos de 10s).\n")
	fmt.Println("💡 Escribe 'ayuda' o consulta tu 'saldo' para verificar los incrementos.")
	
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
				fmt.Printf("✅ Transacción colocada en Mempool (Enviando %.2f a %s)\n", monto, receptor)
			}
		case "salir":
			fmt.Println("👋 Guardando estado de red y cerrando el nodo. ¡Buen descanso!")
			return
		default:
			fmt.Println("❌ Comando no reconocido.")
		}
	}
}
