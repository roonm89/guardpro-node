package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

const TotalSupply = 100000000
const Ticker = "GuardPro"
const WalletFile = "guardpro.priv"

type Block struct {
	Index     int
	Timestamp string
	Data      string
	PrevHash  string
	Hash      string
	FirmaR    *big.Int
	FirmaS    *big.Int
}

var Blockchain []Block

func CalcularDobleHash(b Block) string {
	record := string(rune(b.Index)) + b.Timestamp + b.Data + b.PrevHash
	primerHash := sha256.Sum256([]byte(record))
	dobleHash := sha256.Sum256(primerHash[:])
	return hex.EncodeToString(dobleHash[:])
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
	err = os.WriteFile(WalletFile, bytesPrivados, 0600)
	if err != nil {
		return nil, "", err
	}
	pubKeyHash := sha256.Sum256(elliptic.Marshal(elliptic.P256(), privKey.PublicKey.X, privKey.PublicKey.Y))
	direccionBilletera := "GP" + hex.EncodeToString(pubKeyHash[:16])
	return privKey, direccionBilletera, nil
}

func ManejarConexion(conn net.Conn) {
	defer conn.Close()
	json.NewEncoder(conn).Encode(Blockchain)
}

func ConectarASemilla(target string) {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		fmt.Printf("❌ Error al conectar al Nodo Semilla en %s: %v\n", target, err)
		return
	}
	defer conn.Close()
	fmt.Println("🔗 [Conexión P2P]: Conectado exitosamente al Nodo Semilla.")

	var cadenaRecibida []Block
	err = json.NewDecoder(conn).Decode(&cadenaRecibida)
	if err != nil {
		fmt.Println("Error al decodificar la blockchain:", err)
		return
	}
	fmt.Printf("📦 [Sincronización]: Blockchain descargada. Bloques actuales: %d\n", len(cadenaRecibida))
}

func main() {
	// Definición de banderas/parámetros de consola
	tipoNodo := flag.String("tipo", "semilla", "Tipo de nodo a ejecutar (semilla, validador, billetera)")
	puerto := flag.String("puerto", "8080", "Puerto para escuchar conexiones locales")
	semillaIP := flag.String("semilla", "190.87.251.234:8080", "Dirección IP del nodo semilla")
	flag.Parse()

	fmt.Printf("--- Nodo Guard Pro 7 (%s) | Rol: %s ---\n", Ticker, *tipoNodo)

	_, direccionBilletera, _ := CargarOGenerarBilletera()
	fmt.Printf("Dirección de Billetera: %s\n\n", direccionBilletera)

	// Lógica según el tipo de nodo asignado automáticamente
	if *tipoNodo == "semilla" {
		genesisBlock := Block{0, time.Now().String(), "Bloque Genesis P2P de GuardPro", "", "", nil, nil}
		genesisBlock.Hash = CalcularDobleHash(genesisBlock)
		Blockchain = append(Blockchain, genesisBlock)

		escuchador, err := net.Listen("tcp", ":"+*puerto)
		if err != nil {
			fmt.Println("Error abriendo puerto:", err)
			return
		}
		defer escuchador.Close()
		fmt.Printf("🚀 [Nodo Semilla Activo]: Escuchando en el puerto %s...\n", *puerto)
		for {
			conexion, err := escuchador.Accept()
			if err != nil {
				continue
			}
			go ManejarConexion(conexion)
		}
	} else {
		// Nodos Validadores o Billeteras se conectan automáticamente al arrancar
		fmt.Printf("🛰️ Iniciando cliente... Conectando al semilla en %s\n", *semillaIP)
		ConectarASemilla(*semillaIP)
		
		// Si es validador, simula quedarse activo reportando uptime
		if *tipoNodo == "validador" {
			fmt.Println("⏳ [Modo Validador]: Manteniendo conexión activa para acumular recompensas por Uptime...")
			for {
				time.Sleep(10 * time.Minute)
			}
		}
	}
}
