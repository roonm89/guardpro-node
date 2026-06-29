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
	"flag"
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
const DBFile = "blockchain.json"
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

func GuardarCadenaEnDisco() {
	bytesData, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(DBFile, bytesData, 0644)
}

func CargarCadenaDesdeDisco() bool {
	if _, err := os.Stat(DBFile); os.IsNotExist(err) {
		return false
	}
	bytesData, err := os.ReadFile(DBFile)
	if err != nil {
		return false
	}
	err = json.Unmarshal(bytesData, &Blockchain)
	return err == nil
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

func PropagarTransaccion(tx Transaction, target string) {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("TX:"))
	_ = json.NewEncoder(conn).Encode(tx)
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

			prefijo := make([]byte, 3)
			n, err := conn.Read(prefijo)
			if err == nil && string(prefijo[:n]) == "TX:" {
				var txRecibida Transaction
				if err := json.NewDecoder(conn).Decode(&txRecibida); err == nil {
					Mempool = append(Mempool, txRecibida)
					fmt.Printf("\n📥 [MEMPOOL GLOBAL]: Nueva transacción recibida por red de %s por %.2f %s\nguardpro> ", txRecibida.Emisor[:10], txRecibida.Monto, Ticker)
				}
				return
			}

			_ = json.NewEncoder(conn).Encode(Blockchain)
		}(conexion)
	}
}

func ConectarASemilla(target string) {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		fmt.Printf("\n❌ Error de conexión al servidor semilla en %s: %v\n", target, err)
		return
	}
	defer conn.Close()

	var cadenaRecibida []Block
	err = json.NewDecoder(conn).Decode(&cadenaRecibida)
	if err != nil {
		return
	}
	Blockchain = cadenaRecibida
	GuardarCadenaEnDisco()
	fmt.Printf("📦 [Sincronización]: Cadena descargada. Bloques actuales en red: %d\n", len(Blockchain))
}

func RelojUptime(privKey *ecdsa.PrivateKey, destino string, modo string) {
	// ⏱️ RED UNIFICADA: Todos los nodos avanzan al compás macroeconómico oficial de 5 minutos
	intervalo := 5 * time.Minute
	
	cronometro := time.NewTicker(intervalo)
	for range cronometro.C {
		txsParaBloque := Mempool
		Mempool = []Transaction{}
		
		txRecompensa, _ := CrearTransaccion(privKey, "GP_CREADOR", destino, 10.0)
		txsParaBloque = append(txsParaBloque, txRecompensa)
		
		var prevHash string
		if len(Blockchain) > 0 {
			prevHash = Blockchain[len(Blockchain)-1].Hash
		} else {
			prevHash = ""
		}

		nuevoBloque := Block{
			Index:         len(Blockchain),
			Timestamp:     time.Now().String(),
			Transacciones: txsParaBloque,
			PrevHash:      prevHash,
			Hash:          "",
			FirmaR:        nil,
			FirmaS:        nil,
		}
		nuevoBloque.Hash = CalcularDobleHash(nuevoBloque)
		Blockchain = append(Blockchain, nuevoBloque)
		GuardarCadenaEnDisco()
		
		fmt.Printf("\n🪙 [RECOMPENSA]: +10.00 %s acreditados por Uptime. Bloque #%d minado.\nguardpro> ", Ticker, nuevoBloque.Index)
	}
}

func main() {
	tipoNodo := flag.String("tipo", "semilla", "Tipo de nodo (semilla o validador)")
	semillaIP := flag.String("semilla", "20.226.10.105:8080", "Dirección IP del nodo semilla")
	flag.Parse()

	fmt.Printf("=== CORE OFICIAL UNIFICADO GUARD PRO 7 (%s) ===\n", Ticker)
	fmt.Printf("🎯 Modo de ejecución: [%s]\n", *tipoNodo)
	
	privateKey, miDireccion, _ := CargarOGenerarBilletera()
	fmt.Printf("🔑 Billetera Local: %s\n", miDireccion)

	if CargarCadenaDesdeDisco() {
		fmt.Printf("📂 Base de datos localizada. Libro contable restaurado con %d bloques.\n", len(Blockchain))
	} else {
		fmt.Println("📦 No se encontró historial en disco. Inicializando cadena limpia...")
		if *tipoNodo == "semilla" {
			genesisBlock := Block{0, time.Now().String(), []Transaction{}, "", "", nil, nil}
			genesisBlock.Hash = CalcularDobleHash(genesisBlock)
			Blockchain = append(Blockchain, genesisBlock)

			txInicial, _ := CrearTransaccion(privateKey, "GP_CREADOR", miDireccion, 5000.0)
			Mempool = append(Mempool, txInicial)
			bloque1 := Block{1, time.Now().String(), Mempool, genesisBlock.Hash, "", nil, nil}
			bloque1.Hash = CalcularDobleHash(bloque1)
			Blockchain = append(Blockchain, bloque1)
			Mempool = []Transaction{}
			GuardarCadenaEnDisco()
		}
	}

	go ServidorRed()

	if *tipoNodo == "semilla" {
		go RelojUptime(privateKey, miDireccion, *tipoNodo)
		fmt.Println("🌐 Nodo Semilla escuchando transacciones y pares de forma concurrente.")
	} else {
		fmt.Printf("🛰️ Contactando al Nodo Semilla en la nube: %s...\n", *semillaIP)
		ConectarASemilla(*semillaIP)
		go RelojUptime(privateKey, miDireccion, *tipoNodo)
	}
	
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
			fmt.Println("  suministro  - Muestra el estado de la emisión global de monedas")
			fmt.Println("  cadena      - Despliega todo el historial de bloques en JSON")
			fmt.Println("  enviar      - Realiza una transferencia. Uso: enviar [dirección] [monto]")
			fmt.Println("  mempool     - Muestra las transacciones en sala de espera")
			fmt.Println("  salir       - Apaga el nodo de forma segura")
		case "mempool":
			mempoolJSON, _ := json.MarshalIndent(Mempool, "", "  ")
			fmt.Println(string(mempoolJSON))
		case "suministro":
			emitidas := TotalSupply - ObtenerSaldo("GP_CREADOR")
			restantes := ObtenerSaldo("GP_CREADOR")
			fmt.Printf("📊 ESTADO DEL SUMINISTRO (%s):\n", Ticker)
			fmt.Printf("  ▪️ Suministro Máximo Protegido: %d.00 %s\n", TotalSupply, Ticker)
			fmt.Printf("  ▪️ Monedas Emitidas por Uptime: %.2f %s\n", emitidas, Ticker)
			fmt.Printf("  ▪️ Fondo Común Restante:       %.2f %s\n", restantes, Ticker)
		case "saldo":
			fmt.Printf("💰 Saldo Actual: %.2f %s\n", ObtenerSaldo(miDireccion), Ticker)
		case "cadena":
			cadenaJSON, _ := json.MarshalIndent(Blockchain, "", "  ")
			fmt.Println(string(cadenaJSON))
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
