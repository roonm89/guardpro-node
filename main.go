package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

const TotalSupply = 100000000
const Ticker = "GuardPro"
const WalletFile = "guardpro.priv"
const PuertoRed = ":8080" // Puerto donde escuchará tu nodo semilla

type Block struct {
	Index     int
	Timestamp string
	Data      string
	PrevHash  string
	Hash      string
	FirmaR    *big.Int
	FirmaS    *big.Int
}

// Global para almacenar la cadena en memoria del nodo
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
		fmt.Println("🔑 [Billetera Cargada Exitosamente]")
		return privKey, direccionBilletera, nil
	}

	fmt.Println("✨ [Generando Nueva Billetera]...")
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
	pubKeyHash := sha256.Sum256(elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y))
	direccionBilletera := "GP" + hex.EncodeToString(pubKeyHash[:16])
	return privKey, direccionBilletera, nil
}

// ManejarConexion atiende a otros nodos que se conectan a nuestra laptop
func ManejarConexion(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("🌐 [Nueva Conexión]: Un par (peer) se ha conectado desde %s\n", conn.RemoteAddr().String())

	// Convertir nuestra Blockchain a formato JSON para enviarla por la red
	encoder := json.NewEncoder(conn)
	err := encoder.Encode(Blockchain)
	if err != nil {
		fmt.Println("Error al enviar la blockchain al par:", err)
		return
	}
	fmt.Println("📦 [Sincronización]: Historial de bloques enviado con éxito al nuevo par.")
}

func main() {
	fmt.Printf("--- [Fase 3] Nodo Semilla P2P de Guard Pro 7 (%s) ---\n", Ticker)

	// 1. Cargar Identidad
	_, direccionBilletera, err := CargarOGenerarBilletera()
	if err != nil {
		fmt.Println("Error en billetera:", err)
		return
	}
	fmt.Printf("Dirección del Nodo Semilla: %s\n\n", direccionBilletera)

	// 2. Inicializar la Blockchain con el Bloque Génesis si está vacía
	genesisBlock := Block{0, time.Now().String(), "Bloque Genesis P2P de GuardPro", "", "", nil, nil}
	genesisBlock.Hash = CalcularDobleHash(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)
	fmt.Println("[Bloque 0 - Génesis cargado en memoria local]")

	// 3. Levantar el Servidor P2P de la Blockchain
	escuchador, err := net.Listen("tcp", PuertoRed)
	if err != nil {
		fmt.Printf("Error al abrir el puerto %s: %v\n", PuertoRed, err)
		return
	}
	defer escuchador.Close()
	fmt.Printf("🚀 [Nodo Semilla Activo]: Escuchando conexiones P2P en el puerto %s...\n", PuertoRed)
	fmt.Println("💡 (El nodo se quedará corriendo de forma permanente. Presiona Ctrl+C si deseas apagarlo)\n")

	// Bucle infinito para aceptar conexiones de otros nodos sin detenerse
	for {
		conexion, err := escuchador.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			continue
		}
		// Ejecutar cada conexión en un hilo ligero (Goroutine) para no congelar el nodo
		go ManejarConexion(conexion)
	}
}
